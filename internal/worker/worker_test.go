package worker

import (
	"context"
	"errors"
	"testing"

	"netforge/internal/job"
)

func TestRunWorkers_FiveJobs(t *testing.T) {
	ctx := context.Background()
	repo := job.NewMemoryRepository()

	type seeded struct {
		payload string
		start   job.State
		want    job.State
	}
	plan := []seeded{
		{payload: "ok-1", start: job.Running, want: job.Completed},
		{payload: "ok-2", start: job.Running, want: job.Completed},
		{payload: "ok-3", start: job.Running, want: job.Completed},
		{payload: "bad-payload", start: job.Running, want: job.Failed},
		{payload: "still-waiting", start: job.Waiting, want: job.Waiting},
	}

	created := make([]*job.Job, 0, len(plan))
	for _, item := range plan {
		current, err := repo.CreateJob(ctx, 1, "work", item.payload)
		if err != nil {
			t.Fatalf("create job %q: %v", item.payload, err)
		}
		t.Logf("created %q id=%d state=%s", item.payload, current.ID, current.CurrentState)
		if item.start != job.Waiting {
			if err := repo.UpdateState(ctx, current.ID, item.start); err != nil {
				t.Fatalf("set job %d to %s: %v", current.ID, item.start, err)
			}
			t.Logf("moved %q id=%d to %s", item.payload, current.ID, item.start)
		} else {
			t.Logf("left %q id=%d in %s", item.payload, current.ID, item.start)
		}
		created = append(created, current)
	}

	jobs := make(chan *job.Job, len(created))
	for _, current := range created {
		jobs <- current
		t.Logf("queued %q id=%d", current.Payload, current.ID)
	}
	close(jobs)
	t.Logf("closed channel with %d jobs", len(created))

	handler := func(j *job.Job) error {
		if j.Payload == "bad-payload" {
			t.Logf("handler %q id=%d state=%s -> permanent error", j.Payload, j.ID, j.CurrentState)
			return errors.New("invalid payload")
		}
		t.Logf("handler %q id=%d state=%s -> nil", j.Payload, j.ID, j.CurrentState)
		return nil
	}

	t.Log("starting 2 workers")
	runErr := RunWorkers(ctx, 2, jobs, handler, repo, nil)
	t.Logf("workers returned: %v", runErr)
	if runErr == nil {
		t.Fatal("expected an update error for the waiting job")
	}

	for i, item := range plan {
		got, err := repo.FindByID(ctx, created[i].ID)
		if err != nil {
			t.Fatalf("find job %d: %v", created[i].ID, err)
		}
		t.Logf("result %q id=%d state=%s reason=%q want=%s", item.payload, got.ID, got.CurrentState, got.FailureReason, item.want)
		if got.CurrentState != item.want {
			t.Errorf("job %q: state %s, want %s", item.payload, got.CurrentState, item.want)
		}
	}

	rejected, err := repo.FindByID(ctx, created[4].ID)
	if err != nil {
		t.Fatalf("find waiting job: %v", err)
	}
	if rejected.FailureReason != "Invalid Transition" {
		t.Errorf("waiting job failure reason %q, want invalid transition", rejected.FailureReason)
	}
}

func TestPoolFunction_ClaimsWaitingJobs(t *testing.T) {
	ctx := context.Background()
	repo := job.NewMemoryRepository()

	ids := make([]int64, 0, 5)
	for i := 1; i <= 5; i++ {
		current, err := repo.CreateJob(ctx, 1, "work", "payload-"+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
		if current.CurrentState != job.Waiting {
			t.Fatalf("job %d starts %s, want WAITING", current.ID, current.CurrentState)
		}
		ids = append(ids, current.ID)
	}

	err := PoolFunction(ctx, 2, func(j *job.Job) error { return nil }, repo)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}

	for _, id := range ids {
		got, err := repo.FindByID(ctx, id)
		if err != nil {
			t.Fatalf("find %d: %v", id, err)
		}
		if got.CurrentState != job.Completed {
			t.Errorf("job %d state %s, want COMPLETED", id, got.CurrentState)
		}
	}
}

func TestPoolFunction_CancelBeforeClaim(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := job.NewMemoryRepository()

	current, err := repo.CreateJob(context.Background(), 1, "work", "never-claimed")
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	err = PoolFunction(ctx, 1, func(j *job.Job) error {
		t.Errorf("handler ran for job %d", j.ID)
		return nil
	}, repo)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pool error %v, want context.Canceled", err)
	}

	got, err := repo.FindByID(context.Background(), current.ID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got.CurrentState != job.Waiting {
		t.Errorf("job state %s, want WAITING", got.CurrentState)
	}
}

func TestPoolFunction_CancelBetweenJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	repo := job.NewMemoryRepository()

	ids := make([]int64, 0, 4)
	for i := 0; i < 4; i++ {
		current, err := repo.CreateJob(context.Background(), 1, "work", "payload")
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
		ids = append(ids, current.ID)
	}

	started := make(chan struct{}, 1)
	go func() {
		<-started
		cancel()
	}()

	err := PoolFunction(ctx, 1, func(j *job.Job) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	}, repo)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("pool error %v, want context.Canceled", err)
	}

	var completed, waiting int
	for _, id := range ids {
		got, findErr := repo.FindByID(context.Background(), id)
		if findErr != nil {
			t.Fatalf("find %d: %v", id, findErr)
		}
		switch got.CurrentState {
		case job.Completed:
			completed++
		case job.Waiting:
			waiting++
		default:
			t.Errorf("job %d state %s", id, got.CurrentState)
		}
	}
	if completed != 1 || waiting != 3 {
		t.Fatalf("completed %d, waiting %d, want 1 and 3", completed, waiting)
	}
}
