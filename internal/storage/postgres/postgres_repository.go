package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"netforge/internal/job"
	"time" // pgx driver for database/sql
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func execStmt(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()
	return stmt.ExecContext(ctx, args...)
}

// CreateJob inserts a new job row and returns the created Job.
func (r *Repository) CreateJob(ctx context.Context, runID int64, jobType string, payload string) (*job.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	const q = `
		INSERT INTO jobs (id, run_id, job_type, payload, current_state, failure_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	// ID generation mirrors the in‑memory repo (Unix nano timestamp)
	id := time.Now().UnixNano()
	_, err := execStmt(ctx, r.db, q,
		id,
		runID,
		jobType,
		payload,
		job.Waiting, // initial state
		"None",      // failure_reason default
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert job: %w", err)
	}
	return &job.Job{
		ID:           id,
		RunID:        runID,
		Type:         jobType,
		Payload:      payload,
		CurrentState: job.Waiting,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// FindByID returns a Job by its primary key.
func (r *Repository) FindByID(ctx context.Context, id int64) (*job.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, run_id, job_type, payload, current_state, failure_reason, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	var (
		j           job.Job
		stateStr    string
		failureInfo string
	)
	if err := row.Scan(
		&j.ID,
		&j.RunID,
		&j.Type,
		&j.Payload,
		&stateStr,
		&failureInfo,
		&j.CreatedAt,
		&j.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job %d not found", id)
		}
		return nil, fmt.Errorf("scan job: %w", err)
	}
	st, ok := job.StateFromString(stateStr)
	if !ok {
		return nil, fmt.Errorf("unknown job state %q", stateStr)
	}
	j.CurrentState = st
	j.FailureReason = failureInfo
	return &j, nil
}

// FindByRunID returns all jobs belonging to a given run.
func (r *Repository) FindByRunID(ctx context.Context, runID int64) ([]*job.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, run_id, job_type, payload, current_state, failure_reason, created_at, updated_at
		FROM jobs
		WHERE run_id = $1
		ORDER BY id
	`
	rows, err := r.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("query jobs by run: %w", err)
	}
	defer rows.Close()
	var result []*job.Job
	for rows.Next() {
		var (
			j           job.Job
			stateStr    string
			failureInfo string
		)
		if err := rows.Scan(
			&j.ID,
			&j.RunID,
			&j.Type,
			&j.Payload,
			&stateStr,
			&failureInfo,
			&j.CreatedAt,
			&j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		st, ok := job.StateFromString(stateStr)
		if !ok {
			return nil, fmt.Errorf("unknown job state %q", stateStr)
		}
		j.CurrentState = st
		j.FailureReason = failureInfo
		result = append(result, &j)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return result, nil
}

// UpdateState transitions a job to a new state (validity is checked by the domain logic).
func (r *Repository) UpdateState(ctx context.Context, id int64, next job.State) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		UPDATE jobs
		SET current_state = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := execStmt(ctx, r.db, q, next, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update job state: %w", err)
	}
	return nil
}

// CreateRun inserts a new run row and returns the created RunTask.
func (r *Repository) CreateRun(ctx context.Context, query string) (*job.RunTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	now := time.Now().UTC()
	const q = `
		INSERT INTO runs (id, query, max_jobs, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	id := time.Now().UnixNano()
	_, err := execStmt(ctx, r.db, q,
		id,
		query,
		6,           // default max_jobs (matching InitialiseRun)
		job.Waiting, // initial status
		now,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert run: %w", err)
	}
	return &job.RunTask{
		ID:        id,
		Query:     query,
		MaxJobs:   6,
		Status:    job.Waiting,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// FindRunByID fetches a run by its primary key.
func (r *Repository) FindRunByID(ctx context.Context, id int64) (*job.RunTask, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	const q = `
		SELECT id, query, max_jobs, status, created_at, updated_at
		FROM runs
		WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, q, id)
	var (
		rt        job.RunTask
		statusStr string
	)
	if err := row.Scan(
		&rt.ID,
		&rt.Query,
		&rt.MaxJobs,
		&statusStr,
		&rt.CreatedAt,
		&rt.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run %d not found", id)
		}
		return nil, fmt.Errorf("scan run: %w", err)
	}

	return &rt, nil
}

// UpdateRunStatus updates the status column of a run.
func (r *Repository) UpdateRunStatus(ctx context.Context, id int64, next job.State) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	const q = `
		UPDATE runs
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := execStmt(ctx, r.db, q, next, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	return nil
}

// ClaimJobs atomically marks up to numberOfJobs eligible rows as RUNNING and returns them.
func (r *Repository) ClaimJobs(ctx context.Context, numberOfJobs int) ([]*job.Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if numberOfJobs <= 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	const q = `
		UPDATE jobs
		SET current_state = $2, updated_at = $3
		WHERE id IN (
			SELECT id
			FROM jobs
			WHERE current_state IN ($4, $5)
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		RETURNING id, run_id, job_type, payload, current_state, failure_reason, created_at, updated_at
	`
	rows, err := r.db.QueryContext(ctx, q,
		numberOfJobs,
		job.Running,
		now,
		job.Waiting,
		job.Stalled,
	)
	if err != nil {
		return nil, fmt.Errorf("claim jobs: %w", err)
	}
	defer rows.Close()

	var claimed []*job.Job
	for rows.Next() {
		var (
			j           job.Job
			stateStr    string
			failureInfo string
		)
		if err := rows.Scan(
			&j.ID,
			&j.RunID,
			&j.Type,
			&j.Payload,
			&stateStr,
			&failureInfo,
			&j.CreatedAt,
			&j.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan claimed job: %w", err)
		}
		st, ok := job.StateFromString(stateStr)
		if !ok {
			return nil, fmt.Errorf("unknown job state %q", stateStr)
		}
		j.CurrentState = st
		j.FailureReason = failureInfo
		claimed = append(claimed, &j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("claim jobs rows: %w", err)
	}
	return claimed, nil
}
