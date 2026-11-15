package worker

import (
	"context"
	"errors"
	"netforge/internal/job"
	"sync"
	"time"
)

type Handler func(ctx context.Context, j *job.Job) error

type Store interface {
	UpdateState(ctx context.Context, id int64, next job.State) error
	UpdateAttemptOutcome(ctx context.Context, id int64, next job.State, duration time.Duration) error
}

func RunWorkers(ctx context.Context, n int, jobs <-chan *job.Job, handler Handler, store Store, free chan struct{}, timeout time.Duration) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var updateErrs []error

	addError := func(err error) {
		mu.Lock()
		updateErrs = append(updateErrs, err)
		mu.Unlock()
	}

	signalFree := func() bool {
		if free == nil {
			return true
		}
		select {
		case free <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			for current := range jobs {
				if current == nil {
					addError(errors.New("worker received a nil job"))
					if !signalFree() {
						return
					}
					continue
				}
				if current.CurrentState != job.Running {
					addError(errors.New("worker received a job that is not RUNNING"))
					if !signalFree() {
						return
					}
					continue
				}

				startedAt := time.Now()
				jobCtx, cancel := context.WithTimeout(ctx, timeout)
				err := handler(jobCtx, current)
				cancel()
				attemptDuration := time.Since(startedAt)

				errorName := job.ClassifyError(err)
				var next job.State

				if errorName == job.NoError {
					next = job.Completed
				} else if errorName == job.ErrorPermanent {
					next = job.Failed
				} else if current.AttemptCount > current.RetryLimit {
					next = job.Failed
				} else {
					next = job.Stalled
				}

				if updateErr := store.UpdateAttemptOutcome(context.WithoutCancel(ctx), current.ID, next, attemptDuration); updateErr != nil {
					addError(updateErr)
				}
				if !signalFree() {
					return
				}
			}
		}()
	}
	wg.Wait()
	return errors.Join(updateErrs...)
}
