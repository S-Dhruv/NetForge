package worker

import (
	"context"
	"netforge/internal/job"
)

func Hanlder(ctx context.Context, j * job.Job) error {
	return nil
}

type JobStore struct{
	*job.MemoryRepository
}

func (j *JobStore) UpdateState (ctx context.Context, id int64, next job.State) error{
	TransitionTo()
}