package job

import (
	"context"
	"testing"
)

func TestMemoryRepository_ClaimJobs(t *testing.T) {
	t.Log("setting up three jobs with different retry limits")
	ctx := context.Background()
	repo := NewMemoryRepository()

	runID := int64(1)
	retryLimits := []int64{2, 5, 1}
	for _, retryLimit := range retryLimits {
		_, err := repo.CreateJob(ctx, runID, "test", "payload", retryLimit)
		if err != nil {
			t.Fatalf("create job: %v", err)
		}
	}

	claimed, err := repo.ClaimJobs(ctx, 2)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("expected 2 claimed, got %d", len(claimed))
	}
	t.Logf("first claim returned %d jobs", len(claimed))
	for i, j := range claimed {
		t.Logf("claimed job id=%d state=%s attempt=%d retry-limit=%d", j.ID, j.CurrentState, j.AttemptCount, j.RetryLimit)
		if j.CurrentState != Running {
			t.Errorf("job %d: expected RUNNING, got %s", j.ID, j.CurrentState)
		}
		if j.AttemptCount != 1 {
			t.Errorf("job %d: expected attempt count 1 after first claim, got %d", j.ID, j.AttemptCount)
		}
		if j.RetryLimit != retryLimits[i] {
			t.Errorf("job %d: expected retry limit %d, got %d", j.ID, retryLimits[i], j.RetryLimit)
		}
	}

	all, err := repo.FindByRunID(ctx, runID)
	if err != nil {
		t.Fatalf("find by run: %v", err)
	}
	var waiting int
	for _, j := range all {
		if j.CurrentState == Waiting {
			waiting++
		}
	}
	if waiting != 1 {
		t.Fatalf("expected 1 WAITING job left, got %d", waiting)
	}
	t.Log("confirmed one job remains WAITING")

	// A stalled job is claimable again, and each claim increments its attempt count.
	if err := repo.UpdateState(ctx, claimed[0].ID, Stalled); err != nil {
		t.Fatalf("stall job: %v", err)
	}
	t.Logf("job id=%d moved to STALLED; claiming it again", claimed[0].ID)
	retried, err := repo.ClaimJobs(ctx, 1)
	if err != nil {
		t.Fatalf("reclaim stalled job: %v", err)
	}
	if len(retried) != 1 {
		t.Fatalf("expected one reclaimed job, got %d", len(retried))
	}
	if retried[0].ID != claimed[0].ID {
		t.Fatalf("reclaimed job id %d, want stalled job %d", retried[0].ID, claimed[0].ID)
	}
	if retried[0].AttemptCount != 2 {
		t.Errorf("reclaimed job attempt count %d, want 2", retried[0].AttemptCount)
	}
	t.Logf("reclaimed job id=%d now has attempt=%d", retried[0].ID, retried[0].AttemptCount)

	empty, err := repo.ClaimJobs(ctx, 0)
	if err != nil {
		t.Fatalf("claim zero: %v", err)
	}
	if empty != nil && len(empty) != 0 {
		t.Fatalf("expected nil or empty slice for n<=0, got %v", empty)
	}
	t.Log("zero-capacity claim returned no jobs")
}
