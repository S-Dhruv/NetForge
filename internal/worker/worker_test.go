package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"netforge/internal/job"
)

type failingOutcomeStore struct {
	err error
}

func (s failingOutcomeStore) UpdateState(context.Context, int64, job.State) error {
	return nil
}

func (s failingOutcomeStore) UpdateAttemptOutcome(context.Context, int64, job.State, time.Duration) error {
	return s.err
}

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
		current, err := repo.CreateJob(ctx, 1, "work", item.payload, 5)
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

	handler := func(ctx context.Context, j *job.Job) error {
		if j.Payload == "still-waiting" {
			t.Errorf("handler ran for non-RUNNING job %d", j.ID)
			return nil
		}
		if j.Payload == "bad-payload" {
			t.Logf("handler %q id=%d state=%s -> permanent error", j.Payload, j.ID, j.CurrentState)
			return errors.New("invalid payload")
		}
		t.Logf("handler %q id=%d state=%s -> nil", j.Payload, j.ID, j.CurrentState)
		return nil
	}

	t.Log("starting 2 workers")
	runErr := RunWorkers(ctx, 2, jobs, handler, repo, nil, 3*time.Second)
	t.Logf("workers returned: %v", runErr)
	if runErr == nil {
		t.Fatal("expected an error for the non-RUNNING job")
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

}

func TestPoolFunction_ClaimsWaitingJobs(t *testing.T) {
	ctx := context.Background()
	repo := job.NewMemoryRepository()

	ids := make([]int64, 0, 5)
	for i := 1; i <= 5; i++ {
		current, err := repo.CreateJob(ctx, 1, "work", "payload-"+string(rune('0'+i)), 5)
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
		if current.CurrentState != job.Waiting {
			t.Fatalf("job %d starts %s, want WAITING", current.ID, current.CurrentState)
		}
		ids = append(ids, current.ID)
	}

	err := PoolFunction(ctx, 2, func(ctx context.Context, j *job.Job) error { return nil }, repo, 3*time.Second)
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

	current, err := repo.CreateJob(context.Background(), 1, "work", "never-claimed", 5)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	err = PoolFunction(ctx, 1, func(ctx context.Context, j *job.Job) error {
		t.Errorf("handler ran for job %d", j.ID)
		return nil
	}, repo, 3*time.Second)
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
		current, err := repo.CreateJob(context.Background(), 1, "work", "payload", 5)
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

	err := PoolFunction(ctx, 1, func(ctx context.Context, j *job.Job) error {
		started <- struct{}{}
		<-ctx.Done()
		return nil
	}, repo, 3*time.Second)
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

func TestRunWorkers_DeadlineStallsSlowJob(t *testing.T) {
	ctx := context.Background()
	repo := job.NewMemoryRepository()

	slow, err := repo.CreateJob(ctx, 1, "work", "slow", 5)
	if err != nil {
		t.Fatalf("create slow job: %v", err)
	}
	fast, err := repo.CreateJob(ctx, 1, "work", "fast", 5)
	if err != nil {
		t.Fatalf("create fast job: %v", err)
	}
	for _, id := range []int64{slow.ID, fast.ID} {
		if err := repo.UpdateState(ctx, id, job.Running); err != nil {
			t.Fatalf("set job %d running: %v", id, err)
		}
	}

	jobs := make(chan *job.Job, 2)
	jobs <- slow
	jobs <- fast
	close(jobs)

	err = RunWorkers(ctx, 2, jobs, func(ctx context.Context, j *job.Job) error {
		if j.Payload == "slow" {
			<-ctx.Done()
			return job.ErrNetworkTimeout
		}
		return nil
	}, repo, nil, 30*time.Millisecond)
	if err != nil {
		t.Fatalf("workers: %v", err)
	}

	gotSlow, err := repo.FindByID(ctx, slow.ID)
	if err != nil {
		t.Fatalf("find slow: %v", err)
	}
	if gotSlow.CurrentState != job.Stalled {
		t.Errorf("slow job state %s, want STALLED", gotSlow.CurrentState)
	}

	gotFast, err := repo.FindByID(ctx, fast.ID)
	if err != nil {
		t.Fatalf("find fast: %v", err)
	}
	if gotFast.CurrentState != job.Completed {
		t.Errorf("fast job state %s, want COMPLETED", gotFast.CurrentState)
	}
}

func TestRunWorkers_RecordsLatestAttemptDuration(t *testing.T) {
	t.Log("creating one RUNNING job whose handler sleeps briefly")
	ctx := context.Background()
	repo := job.NewMemoryRepository()
	current, err := repo.CreateJob(ctx, 1, "work", "timed", 5)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := repo.UpdateState(ctx, current.ID, job.Running); err != nil {
		t.Fatalf("set job running: %v", err)
	}

	jobs := make(chan *job.Job, 1)
	jobs <- current
	close(jobs)

	minimumDuration := 10 * time.Millisecond
	err = RunWorkers(ctx, 1, jobs, func(ctx context.Context, j *job.Job) error {
		t.Logf("handler started for job id=%d", j.ID)
		time.Sleep(minimumDuration)
		t.Logf("handler finished for job id=%d", j.ID)
		return nil
	}, repo, nil, time.Second)
	if err != nil {
		t.Fatalf("workers: %v", err)
	}

	got, err := repo.FindByID(ctx, current.ID)
	if err != nil {
		t.Fatalf("find job: %v", err)
	}
	if got.LatestAttemptDuration < minimumDuration {
		t.Errorf("latest attempt duration %s, want at least %s", got.LatestAttemptDuration, minimumDuration)
	}
	t.Logf("stored state=%s latest-duration=%s", got.CurrentState, got.LatestAttemptDuration)
}

func TestRunWorkers_OutcomeBranches(t *testing.T) {
	temporaryErr := job.ErrNetworkTimeout
	permanentErr := errors.New("invalid payload")
	tests := []struct {
		name         string
		retryLimit   int64
		attemptCount int64
		handlerErr   error
		wantState    job.State
	}{
		{name: "success on first attempt", retryLimit: 5, attemptCount: 1, wantState: job.Completed},
		{name: "success on final attempt", retryLimit: 5, attemptCount: 6, wantState: job.Completed},
		{name: "permanent error fails immediately", retryLimit: 5, attemptCount: 1, handlerErr: permanentErr, wantState: job.Failed},
		{name: "temporary error leaves retries", retryLimit: 5, attemptCount: 5, handlerErr: temporaryErr, wantState: job.Stalled},
		{name: "temporary error exhausts retries", retryLimit: 5, attemptCount: 6, handlerErr: temporaryErr, wantState: job.Failed},
		{name: "zero retries fails on initial error", retryLimit: 0, attemptCount: 1, handlerErr: temporaryErr, wantState: job.Failed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("starting case: retries=%d attempt=%d handler-error=%v want-state=%s", tt.retryLimit, tt.attemptCount, tt.handlerErr, tt.wantState)
			ctx := context.Background()
			repo := job.NewMemoryRepository()
			current, err := repo.CreateJob(ctx, 1, "work", "payload", tt.retryLimit)
			if err != nil {
				t.Fatalf("create job: %v", err)
			}
			current.AttemptCount = tt.attemptCount
			if err := repo.UpdateState(ctx, current.ID, job.Running); err != nil {
				t.Fatalf("set job running: %v", err)
			}

			jobs := make(chan *job.Job, 1)
			jobs <- current
			close(jobs)
			handlerCalls := 0
			err = RunWorkers(ctx, 1, jobs, func(context.Context, *job.Job) error {
				handlerCalls++
				t.Logf("handler called for attempt %d", tt.attemptCount)
				return tt.handlerErr
			}, repo, nil, time.Second)
			if err != nil {
				t.Fatalf("workers: %v", err)
			}
			if handlerCalls != 1 {
				t.Fatalf("handler called %d times, want 1", handlerCalls)
			}

			got, err := repo.FindByID(ctx, current.ID)
			if err != nil {
				t.Fatalf("find job: %v", err)
			}
			if got.CurrentState != tt.wantState {
				t.Errorf("state %s, want %s", got.CurrentState, tt.wantState)
			}
			t.Logf("case finished with state=%s attempt=%d duration=%s", got.CurrentState, got.AttemptCount, got.LatestAttemptDuration)
		})
	}
}

func TestRunWorkers_RejectsNilJob(t *testing.T) {
	t.Log("sending nil job; handler should be skipped and worker should report the input error")
	jobs := make(chan *job.Job, 1)
	jobs <- nil
	close(jobs)
	handlerCalled := false
	err := RunWorkers(context.Background(), 1, jobs, func(context.Context, *job.Job) error {
		handlerCalled = true
		return nil
	}, job.NewMemoryRepository(), nil, time.Second)
	if err == nil {
		t.Fatal("expected nil job to be reported")
	}
	if handlerCalled {
		t.Fatal("handler ran for nil job")
	}
}

func TestRunWorkers_ReturnsOutcomePersistenceError(t *testing.T) {
	t.Log("forcing outcome persistence to fail and checking the worker returns that error")
	ctx := context.Background()
	repo := job.NewMemoryRepository()
	current, err := repo.CreateJob(ctx, 1, "work", "payload", 5)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := repo.UpdateState(ctx, current.ID, job.Running); err != nil {
		t.Fatalf("set job running: %v", err)
	}

	wantErr := errors.New("write failed")
	jobs := make(chan *job.Job, 1)
	jobs <- current
	close(jobs)
	err = RunWorkers(ctx, 1, jobs, func(context.Context, *job.Job) error {
		return nil
	}, failingOutcomeStore{err: wantErr}, nil, time.Second)
	if !errors.Is(err, wantErr) {
		t.Fatalf("workers error %v, want %v", err, wantErr)
	}
}

func TestRunWorkers_StopsAfterContextCancellationWhileSignaling(t *testing.T) {
	t.Log("cancelling the pool context before the worker tries to signal a free slot")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := job.NewMemoryRepository()
	current, err := repo.CreateJob(context.Background(), 1, "work", "payload", 5)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if err := repo.UpdateState(context.Background(), current.ID, job.Running); err != nil {
		t.Fatalf("set job running: %v", err)
	}
	jobs := make(chan *job.Job, 1)
	jobs <- current
	close(jobs)

	// With no receiver, the free-slot send cannot proceed; the cancelled context
	// lets the worker stop instead of blocking forever.
	err = RunWorkers(ctx, 1, jobs, func(ctx context.Context, _ *job.Job) error {
		<-ctx.Done()
		return job.ErrNetworkTimeout
	}, repo, make(chan struct{}), time.Second)
	if err != nil {
		t.Fatalf("workers: %v", err)
	}
}
