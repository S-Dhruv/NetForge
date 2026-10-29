CREATE TABLE IF NOT EXISTS runs (
    id BIGINT PRIMARY KEY,
    query TEXT NOT NULL,
    max_jobs INT NOT NULL DEFAULT 6,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT PRIMARY KEY,
    run_id BIGINT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    job_type VARCHAR(100) NOT NULL,
    payload TEXT NOT NULL,
    current_state VARCHAR(50) NOT NULL,
    failure_reason TEXT NOT NULL DEFAULT 'None',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_run_id ON jobs(run_id);
CREATE INDEX IF NOT EXISTS idx_jobs_current_state ON jobs(current_state);