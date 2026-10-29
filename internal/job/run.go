package job

import (
	"errors"
	"time"
)

type RunTask struct {
	ID        int64
	Query     string
	MaxJobs   int32
	Status    State
	CreatedAt time.Time
	UpdatedAt time.Time
}

func InitializeRun(query string) (*RunTask, error) {

	if query == "" {
		return nil, errors.New("Invalid query")
	}

	return &RunTask{
		ID:        time.Now().UnixNano(),
		Query:     query,
		MaxJobs:   6, // hardcoded for now
		Status:    Waiting,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil

}
