package job

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryRepository_CreateJob(t *testing.T) {
	tests := []struct {
		name    string
		runId   int64
		jobType string
		payload string
	}{
		{name: "create job", runId: 1, jobType: "creation", payload: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			memoryRepo := NewMemoryRepository()

			got, err := memoryRepo.CreateJob(ctx, tt.runId, tt.jobType, tt.payload)
			if err != nil {
				t.Fatalf("unexpected error creating job: %v", err)
			}

			if got == nil || got.ID == 0 {
				t.Fatalf("expected valid job with non-zero ID, got %v", got)
			}

			found, err := memoryRepo.FindByID(ctx, got.ID)
			if err != nil {
				t.Fatalf("expected to find stored job with ID %d, got error: %v", got.ID, err)
			}

			if found.Type != tt.jobType {
				t.Errorf("got job type %q, want %q", found.Type, tt.jobType)
			}

			if found.Payload != tt.payload {
				t.Errorf("got payload %q, want %q", found.Payload, tt.payload)
			}
		})
	}
}

func TestMemoryRepository_FindByRunID(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	// Seed 2 jobs for Run 100, and 1 job for Run 200
	_, err := repo.CreateJob(ctx, 100, "discovery", "url1")
	if err != nil {
		t.Fatalf("failed to create job 1: %v", err)
	}
	_, err = repo.CreateJob(ctx, 100, "fetch", "url2")
	if err != nil {
		t.Fatalf("failed to create job 2: %v", err)
	}
	_, err = repo.CreateJob(ctx, 200, "analyze", "url3")
	if err != nil {
		t.Fatalf("failed to create job 3: %v", err)
	}

	// Query Run 100
	jobsRun100, err := repo.FindByRunID(ctx, 100)
	if err != nil {
		t.Fatalf("unexpected error finding by runID 100: %v", err)
	}
	if len(jobsRun100) != 2 {
		t.Errorf("expected 2 jobs for runID 100, got %d", len(jobsRun100))
	}

	// Query Run 200
	jobsRun200, err := repo.FindByRunID(ctx, 200)
	if err != nil {
		t.Fatalf("unexpected error finding by runID 200: %v", err)
	}
	if len(jobsRun200) != 1 {
		t.Errorf("expected 1 job for runID 200, got %d", len(jobsRun200))
	}
}

func TestMemoryRepository_UpdateState(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	job, err := repo.CreateJob(ctx, 1, "fetch", "data")
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// 1. Valid transition: Waiting -> Running
	if err := repo.UpdateState(ctx, job.ID, Running); err != nil {
		t.Fatalf("expected valid transition to Running, got: %v", err)
	}

	found, err := repo.FindByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to find job: %v", err)
	}
	if found.CurrentState != Running {
		t.Errorf("expected state %v, got %v", Running, found.CurrentState)
	}

	// 2. Invalid transition: Running -> Waiting (must return error)
	if err := repo.UpdateState(ctx, job.ID, Waiting); err == nil {
		t.Errorf("expected error transitioning from Running to Waiting, got nil")
	}

	// 3. Updating non-existent job ID
	if err := repo.UpdateState(ctx, 9999999, Running); err == nil {
		t.Errorf("expected error updating non-existent job ID, got nil")
	}
}

// I didnt write the below code I have no clue what it means (will get to understanding of it after learning more about go routines) TODO: Learn go routines and concurrency
func TestMemoryRepository_ConcurrentRace(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	var wg sync.WaitGroup
	numWorkers := 10
	numJobsPerWorker := 20

	// Concurrently create jobs across multiple goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runID := int64(workerID + 1)
			for j := 0; j < numJobsPerWorker; j++ {
				job, err := repo.CreateJob(ctx, runID, "task", "payload")
				if err != nil {
					t.Errorf("concurrent CreateJob error: %v", err)
					return
				}
				// Concurrently read and update
				_, _ = repo.FindByID(ctx, job.ID)
				_ = repo.UpdateState(ctx, job.ID, Running)
			}
		}(i)
	}

	wg.Wait()

	// Verify all jobs were safely stored without data corruption
	totalExpected := numWorkers * numJobsPerWorker
	allJobs := make([]*Job, 0)
	for i := 0; i < numWorkers; i++ {
		jobs, err := repo.FindByRunID(ctx, int64(i+1))
		if err != nil {
			t.Fatalf("unexpected error finding run: %v", err)
		}
		allJobs = append(allJobs, jobs...)
	}

	if len(allJobs) != totalExpected {
		t.Errorf("expected %d total jobs created, got %d", totalExpected, len(allJobs))
	}
}
