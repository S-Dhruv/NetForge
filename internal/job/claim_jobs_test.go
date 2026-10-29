package job

import (
	"context"
	"testing"
)

func TestMemoryRepository_ClaimJobs(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	runID := int64(1)
	for i := 0; i < 3; i++ {
		_, err := repo.CreateJob(ctx, runID, "test", "payload")
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
	for _, j := range claimed {
		if j.CurrentState != Running {
			t.Errorf("job %d: expected RUNNING, got %s", j.ID, j.CurrentState)
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

	empty, err := repo.ClaimJobs(ctx, 0)
	if err != nil {
		t.Fatalf("claim zero: %v", err)
	}
	if empty != nil && len(empty) != 0 {
		t.Fatalf("expected nil or empty slice for n<=0, got %v", empty)
	}
}
