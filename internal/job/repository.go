package job

import "context"

type Repository interface{
	FindByID(ctx context.Context, id int64) (*Job, error)
	FindByRunID(ctx context.Context, runId int64) ([]*Job ,error)
	CreateJob(ctx context.Context, runId int64, jobType string, payload string) (*Job, error)
	UpdateState(ctx context.Context, id int64, next State) (error)
}


