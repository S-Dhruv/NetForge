package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"netforge/internal/job"
	"netforge/internal/storage/postgres"
	"netforge/internal/worker"
	"time"
)

func NewPostgresRepo() (*postgres.Repository, error) {
	dsn := os.Getenv("DATABASE_URL") // e.g. postgres://netforge:netforge_secret@localhost:5453/netforge?sslmode=disable
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	// optional: ping to verify connection
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return postgres.New(db), nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	repo, err := NewPostgresRepo()
	if err != nil {
		log.Fatalf("failed to create repo: %v", err)
	}

	// 1. Create a RunTask
	// run, err := repo.CreateRun(ctx, "SELECT * FROM example")
	// if err != nil {
	// 	log.Fatalf("create run failed: %v", err)
	// }
	// log.Printf("Created run: %+v", run)

	// // 2. Create a Job belonging to the run
	// job1, err := repo.CreateJob(ctx, run.ID, "example-job", "payload-data")
	// if err != nil {
	// 	log.Fatalf("create job failed: %v", err)
	// }
	// log.Printf("Created job: %+v", job1)

	// // 3. Transition job state to Running
	// if err := repo.UpdateState(ctx, job1.ID, job.Running); err != nil {
	// 	log.Fatalf("update job state failed: %v", err)
	// }
	// // fetch updated job to verify
	// updatedJob, err := repo.FindByID(ctx, job1.ID)
	// if err != nil {
	// 	log.Fatalf("find job failed: %v", err)
	// }
	// log.Printf("Job after state change: %+v", updatedJob)

	// // 4. List all jobs for the run
	// jobs, err := repo.FindByRunID(ctx, run.ID)
	// if err != nil {
	// 	log.Fatalf("list jobs failed: %v", err)
	// }
	// log.Printf("All jobs for run %d: %d found", run.ID, len(jobs))

	// // 5. Update run status to Running
	// if err := repo.UpdateRunStatus(ctx, run.ID, job.Running); err != nil {
	// 	log.Fatalf("update run status failed: %v", err)
	// }
	// updatedRun, err := repo.FindRunByID(ctx, run.ID)
	// if err != nil {
	// 	log.Fatalf("find run failed: %v", err)
	// }
	// log.Printf("Run after status change: %+v", updatedRun)

	run1, err := repo.CreateRun(ctx, "SELECT * from example")
	if err != nil {
		log.Fatalf("create run failed: %v", err)
	}
	log.Printf("Created run %+v", run1)

	run2, err := repo.CreateRun(ctx, "SELECT * from example")
	if err != nil {
		log.Fatalf("create run failed: %v", err)
	}
	log.Printf("Created run %+v", run2)

	run3, err := repo.CreateRun(ctx, "SELECT * from example")
	if err != nil {
		log.Fatalf("create run failed: %v", err)
	}
	log.Printf("Created run %+v", run3)

	job2, err := repo.CreateJob(ctx, run1.ID, "test-job-context", "payload-jargon", 5)
	if err != nil {
		log.Fatalf("create job failed: %v", err)
	}
	log.Printf("Job created : %+v", job2)

	job3, err := repo.CreateJob(ctx, run2.ID, "test-job-context", "payload-jargon", 5)
	if err != nil {
		log.Fatalf("create job failed: %v", err)
	}
	log.Printf("Job created : %+v", job3)

	job4, err := repo.CreateJob(ctx, run3.ID, "test-job-context", "payload-jargon", 5)
	if err != nil {
		log.Fatalf("create job failed: %v", err)
	}
	log.Printf("Job created : %+v", job4)

	err = worker.PoolFunction(ctx, 3, handler, repo, 3*time.Second)
	if err != nil {
		log.Fatalf("Pool function failed : %v", err)
	}
	log.Printf("Pool function executed")

	updatedRun1, err := repo.FindRunByID(ctx, run1.ID)
	if err != nil {
		log.Fatalf("find run failed: %v", err)
	}
	log.Printf("Run after status change: %+v", updatedRun1)

	updatedRun2, err := repo.FindRunByID(ctx, run2.ID)
	if err != nil {
		log.Fatalf("find run failed: %v", err)
	}
	log.Printf("Run after status change: %+v", updatedRun2)

	updatedRun3, err := repo.FindRunByID(ctx, run3.ID)
	if err != nil {
		log.Fatalf("find run failed: %v", err)
	}
	log.Printf("Run after status change: %+v", updatedRun3)

}

func handler(ctx context.Context, j *job.Job) error {
	select {
	case <-ctx.Done():
		return job.ErrNetworkTimeout
	case <-time.After(5 * time.Second):
		return nil
	}
}
