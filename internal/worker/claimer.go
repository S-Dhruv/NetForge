package worker

import (
	"context"
	"errors"
	"netforge/internal/job"
)

type ClaimStore interface {
	Store
	ClaimJobs(ctx context.Context, numJobs int) ([]*job.Job, error)
}

// ClaimStore is the repository surface the pool needs. RunWorkers still sees only Store.

func PoolFunction(ctx context.Context, n int, handler Handler, repo ClaimStore) error {
	if n <= 0 {
		return errors.New("worker count must be positive")
	}

	jobs := make(chan *job.Job)
	free := make(chan struct{}, n)
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunWorkers(ctx, n, jobs, handler, repo, free)
	}()

	slots := n
	inFlight := 0
	for {
		if err := ctx.Err(); err != nil {
			close(jobs)
			return errors.Join(err, <-errCh)
		}
		if slots == 0 {
			select {
			case <-ctx.Done():
				close(jobs)
				return errors.Join(ctx.Err(), <-errCh)
			case <-free:
				slots++
				inFlight--
			}
			continue
		}

		claimed, err := repo.ClaimJobs(ctx, slots)
		if err != nil {
			close(jobs)
			return errors.Join(err, <-errCh)
		}
		if len(claimed) == 0 {
			if inFlight == 0 {
				close(jobs)
				return <-errCh
			}
			slots = 0
			continue
		}

		for _, current := range claimed {
			select {
			case <-ctx.Done():
				close(jobs)
				return errors.Join(ctx.Err(), <-errCh)
			case jobs <- current:
				slots--
				inFlight++
			}
		}
	}
}
