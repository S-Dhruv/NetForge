package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"netforge/internal/job"
	"netforge/internal/storage/postgres"
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
	ctx := context.Background()
	repo, err := NewPostgresRepo()
	if err != nil {
		log.Fatalf("failed to create repo: %v", err)
	}

	// 1. Create a RunTask
	run, err := repo.CreateRun(ctx, "SELECT * FROM example")
	if err != nil {
		log.Fatalf("create run failed: %v", err)
	}
	log.Printf("Created run: %+v", run)

	// 2. Create a Job belonging to the run
	job1, err := repo.CreateJob(ctx, run.ID, "example-job", "payload-data")
	if err != nil {
		log.Fatalf("create job failed: %v", err)
	}
	log.Printf("Created job: %+v", job1)

	// 3. Transition job state to Running
	if err := repo.UpdateState(ctx, job1.ID, job.Running); err != nil {
		log.Fatalf("update job state failed: %v", err)
	}
	// fetch updated job to verify
	updatedJob, err := repo.FindByID(ctx, job1.ID)
	if err != nil {
		log.Fatalf("find job failed: %v", err)
	}
	log.Printf("Job after state change: %+v", updatedJob)

	// 4. List all jobs for the run
	jobs, err := repo.FindByRunID(ctx, run.ID)
	if err != nil {
		log.Fatalf("list jobs failed: %v", err)
	}
	log.Printf("All jobs for run %d: %d found", run.ID, len(jobs))

	// 5. Update run status to Running
	if err := repo.UpdateRunStatus(ctx, run.ID, job.Running); err != nil {
		log.Fatalf("update run status failed: %v", err)
	}
	updatedRun, err := repo.FindRunByID(ctx, run.ID)
	if err != nil {
		log.Fatalf("find run failed: %v", err)
	}
	log.Printf("Run after status change: %+v", updatedRun)
}
