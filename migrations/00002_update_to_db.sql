-- Apply after 00001_init.up.sql. Keep already-applied migrations unchanged.
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS retry_limit BIGINT NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS attempt_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS latest_attempt_duration BIGINT NOT NULL DEFAULT 0;

-- Duration is stored as time.Duration nanoseconds, not as a timestamp.
ALTER TABLE jobs
    ALTER COLUMN latest_attempt_duration TYPE BIGINT USING 0,
    ALTER COLUMN latest_attempt_duration SET DEFAULT 0,
    ALTER COLUMN latest_attempt_duration SET NOT NULL;
