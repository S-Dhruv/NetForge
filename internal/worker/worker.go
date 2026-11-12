package worker

import (
	"context"
	"errors"
	"netforge/internal/job"
	"sync"
)

type Handler func(j *job.Job) error

type Store interface {
	UpdateState(ctx context.Context, id int64, next job.State) error
}

func RunWorkers(ctx context.Context, n int, jobs <-chan *job.Job, handler Handler, store Store, free chan struct{}) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var updateErrs []error

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for current := range jobs {
				err := handler(current)
				errorName := job.ClassifyError(err)
				var next job.State

				if errorName == job.NoError {
					next = job.Completed
				} else if errorName == job.ErrorPermanent {
					next = job.Failed
				} else {
					next = job.Stalled
				}

				if updateErr := store.UpdateState(context.WithoutCancel(ctx), current.ID, next); updateErr != nil {
					mu.Lock()
					updateErrs = append(updateErrs, updateErr)
					mu.Unlock()
				}
				if free != nil {
					select {
					case free <- struct{}{}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	return errors.Join(updateErrs...)
}
