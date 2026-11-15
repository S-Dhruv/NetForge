ALTER TABLE jobs
    DROP COLUMN IF EXISTS latest_attempt_duration,
    DROP COLUMN IF EXISTS attempt_count,
    DROP COLUMN IF EXISTS retry_limit;
