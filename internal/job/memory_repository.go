package job

import (
	"context"
	"errors"
	"sync"
)

type MemoryRepository struct {
	jobs map[int64]*Job
	mu   sync.RWMutex
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		jobs: make(map[int64]*Job),
	}
}

func (r *MemoryRepository) FindByID(ctx context.Context, id int64) (*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	currentJob := r.jobs[id]
	defer r.mu.RUnlock()

	if currentJob == nil {
		return nil, errors.New("No ID entered")
	}

	return currentJob, nil
}

func (r *MemoryRepository) FindByRunID(ctx context.Context, runId int64) ([]*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	allJobs := make([]*Job, 0)

	r.mu.RLock()
	for _, jobDetails := range r.jobs {
		if jobDetails.RunID == runId {
			allJobs = append(allJobs, jobDetails)
		}
	}
	r.mu.RUnlock()

	return allJobs, nil
}

func (r *MemoryRepository) CreateJob(ctx context.Context, runId int64, jobType string, payload string) (*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if runId == 0 || jobType == "" || payload == "" {
		return nil, errors.New("Missing fields")
	}

	newJob := NewJob(runId, jobType, payload)
	if newJob == nil {
		return nil, errors.New("Issue with job creation")
	}

	r.mu.Lock()
	r.jobs[newJob.ID] = newJob
	r.mu.Unlock()

	return newJob, nil
}

func (r *MemoryRepository) UpdateState(ctx context.Context, id int64, next State) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	
	currentJob := r.jobs[id]

	if currentJob == nil {
		return errors.New("No current Job found")
	}

	return currentJob.TransitionTo(next)
}
