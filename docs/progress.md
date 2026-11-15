# NetForge — Progress Log

Append-only, dated record of completed tasks, tests actually run with verified outcomes, decisions, blockers, and the next concrete task.

---

### 2026-10-20
- **Task:** Initial project planning and environment inspection.
- **Decision:** Accepted ADR-001 (Select Go `1.27+` as core implementation language).
- **Environment:** Linux x86_64, Go 1.27.2 verified installed, Python 3.14.8 installed.
- **Outcome:** Clean setup confirmed.

---

### 2026-10-23
- **Milestone:** M0 (Complete) & M1 (In Progress).
- **Specification:** Received and integrated the authoritative project guide (`docs/SPECIFICATION.md` from `NetForge_Project_and_Learning_Guide.docx`).
- **Work Completed:**
  - Initialized Go module `netforge` in project root (`go.mod`).
  - Learner designed and implemented core lifecycle states and transition logic in `internal/job/state.go` (`Waiting`, `Running`, `Stalled`, `Completed`, `Failed`).
  - Learner implemented idiomatic table-driven unit test suite in `internal/job/state_test.go`.
  - Learner designed and implemented `Job` domain entity and `TransitionTo` method in `internal/job/job.go`.
  - Learner implemented error categories and `ClassifyError` in `internal/job/errors.go`.
  - Learner defined domain storage contract `Repository` interface with `context.Context` parameters in `internal/job/repository.go`.
  - Created standardized memory and continuity files per Section 15 of specification (`milestones.md`, `progress.md`, `teaching-notes.md`, `architecture.md`, `decisions/`, `.env.example`, root `README.md`).
- **Tests & Commands Run:**
  - `go build ./...` -> Exited 0 (clean compilation).
  - `go test -count=1 -v ./internal/job` -> All 3 subtests passed:
    - `TestCanTransition/waiting_to_running_is_valid` (PASS)
    - `TestCanTransition/waiting_to_completed_is_invalid` (PASS)
    - `TestCanTransition/waiting_to_stalled_is_invalid_failing_on_purpose` (PASS - verified negative transition).
### 2026-10-24
- **Milestone:** M1 (In Progress).
- **Work Completed:**
  - Designed and implemented `MemoryRepository` struct satisfying `job.Repository` in `internal/job/memory_repository.go`.
  - Implemented `NewMemoryRepository` constructor, initializing internal `jobs` map.
  - Implemented all 4 interface methods: `FindByID`, `FindByRunID`, `CreateJob`, and `UpdateState`.
  - Integrated `ctx.Err()` cancellation handling across all repository methods.
  - Resolved compiler issues (unused variables in range loops, return type consistency).
- **Tests & Commands Run:**
  - `go test -v ./internal/job` -> All 3 subtests in `state_test.go` pass (exit code 0).
  - Compilation of `memory_repository.go` verified.
### 2026-10-25
- **Milestone:** M1 (In Progress).
- **Work Completed:**
  - Implemented comprehensive test suite in `internal/job/memory_repository_test.go`:
    - `TestMemoryRepository_CreateJob`: creation and retrieval integrity.
    - `TestMemoryRepository_FindByRunID`: fan-out multi-job querying by run ID.
    - `TestMemoryRepository_UpdateState`: valid transitions, invalid transition error rejection, and non-existent ID error handling.
    - `TestMemoryRepository_ConcurrentRace`: 10 concurrent goroutines executing 200 total jobs with simultaneous read and write operations.
  - Designed and implemented `RunTask` entity and `InitializeRun` constructor with input validation in `internal/job/run.go`.
  - Configured local containerized PostgreSQL 16 in `docker-compose.yml` and environment settings in `.env`.
  - Downloaded official Go driver `github.com/jackc/pgx/v5`.
  - Created versioned SQL migration scripts `migrations/00001_init.up.sql` (defining `runs` and `jobs` tables with foreign keys and B-tree indexes) and `migrations/00001_init.down.sql`.
  - Architecture decision: isolate persistence in `internal/storage/postgres/` implementing `job.Repository` to maintain clean domain boundaries.
- **Tests & Commands Run:**
  - `go test -race -v ./internal/job` -> All 4 repository tests and 3 state transition tests passed with **zero data races** (exit code 0).
  - `go build ./...` -> Exited 0.
- **Next Task:**
  - Implement `internal/storage/postgres/repository.go` implementing `job.Repository` using `database/sql` and `pgx`.
  - Write integration tests against PostgreSQL verifying durable persistence across restarts.

---

### 2026-10-25 (continued)
- **Work Completed:**
  - Implemented full PostgreSQL repository in `internal/storage/postgres/postgres_repository.go`:
    - All 4 job methods: `CreateJob`, `FindByID`, `FindByRunID`, `UpdateState`.
    - All 3 run methods: `CreateRun`, `FindRunByID`, `UpdateRunStatus`.
    - Used `database/sql` with `pgx/v5/stdlib` driver via blank import.
    - Added `StateFromString` helper to `internal/job/state.go` for DB round-trips.
  - Rewrote `main.go` as `package main` with a `main()` function demonstrating the full CRUD lifecycle against PostgreSQL.
  - Applied SQL migrations to running Docker container.
- **Blockers & Fixes:**
  - `go mod tidy` (run in sandbox without network) wiped `go.sum` and dependencies. Fixed by running `go get github.com/jackc/pgx/v5@v5.11.0` and `go mod tidy` from a terminal with internet access.
  - `sql: unknown driver "pgx"` — fixed by adding `_ "github.com/jackc/pgx/v5/stdlib"` blank import in `main.go`.
- **Verified Commands:**
  - `go build ./...` → exit 0 (clean build, all packages compile).
- **Next Task:**
  - Apply migrations to running container: `docker exec -i netforge-postgres psql -U netforge -d netforge < migrations/00001_init.up.sql`
  - Run `main.go` against PostgreSQL to verify end-to-end persistence.
  - Write integration tests in `internal/storage/postgres/postgres_repository_test.go` mirroring the memory repository test suite.
  - Verify jobs survive a Docker container restart (milestone M1 acceptance gate).

---

### 2026-10-25 (M1 closed)
- **Verified:** `docker compose stop && docker compose start` → `SELECT id, status FROM runs` and `SELECT id, current_state FROM jobs` both returned `RUNNING`. Data survived restart.
- **M1 Acceptance Gate: MET.**
- **Next:** M2 — Single-Process Worker Loop.

---

### 2026-10-26 — M2 Design
- **Decisions made (M2 worker loop design):**
  - Batch claiming: fetch `N` jobs at once using `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N` where N = free workers. Re-query only when in-memory list is empty.
  - Claim eligible states: `WAITING` and `STALLED` (stalled jobs are retry-eligible).
  - Bounded concurrency: buffered channel as semaphore (capacity = MaxJobs). Each worker acquires a slot before starting a job, releases on completion. No separate registry or mutex needed — the channel tracks concurrency naturally.
  - Graceful shutdown: `context.Context` cancellation signals workers to stop between jobs (not mid-execution). `sync.WaitGroup` ensures main waits for all goroutines to finish.
  - Signal handling: `SIGINT`/`SIGTERM` triggers context cancellation (to be added).
- **Next Task:** Implement `ClaimJobs(ctx, n int)` method on PostgreSQL repository, then the worker loop in `internal/worker/`.

---

### 2026-10-26 — Tutor orientation
- **Task:** Consolidated project + pedagogy for LLM tutors.
- **Work Completed:**
  - Added [`docs/llm-tutor-brief.md`](llm-tutor-brief.md) (session entry point: teaching contract, M2 design, repo map, anti-patterns).
  - Linked brief from [`docs/teaching-notes.md`](teaching-notes.md).
  - Synced [`README.md`](../README.md) milestone status (M1 complete, M2 in progress).
- **Tests & Commands Run:**
  - `go test -count=1 ./...` → `internal/job` ok; `postgres` package has no tests yet.
- **Next Task:** Implement `ClaimJobs` on Postgres, then `internal/worker/` (learner-owned) per M2 design above.

---

### 2026-10-30 — Session handoff
- **State:** Postgres + memory `ClaimJobs` done; `claim_jobs_test.go` kept; **`internal/worker/` not started** (learner-owned).
- **Doc:** Added [`next-session.md`](next-session.md) — slices 2→4, M2 design recap, tutor contract for new chats.
- **Next task (learner):** M2 **slice 2** — workers + job channel + `UpdateState`; no claimer; see `next-session.md`.

---

### 2026-11-12 — M2 slice 2 workers
- **Work:** `internal/worker` `Handler`, narrow `Store`, and `RunWorkers` (learner). Test `TestRunWorkers_FiveJobs` sends 5 jobs to 2 workers: 3 succeed, 1 handler rejects a bad payload, 1 stays `WAITING` so `RUNNING → COMPLETED` is illegal.
- **Tests:** `go test -count=1 -v ./internal/worker` → PASS. `go test -race -count=1 ./internal/worker` → PASS.
- **Gap closed:** `RunWorkers` now returns joined `UpdateState` errors and `continue`s to the next job. The five-job test still passes: four jobs reach `COMPLETED`/`FAILED`, the `WAITING` job reports `ErrInvalidTranistion`, and the pool does not stop.
- **Next task:** M2 slice 3 — central claimer. Workers stay consumers only.

---

### 2026-11-12 — M2 slice 3 pool
- **Work:** `PoolFunction` starts `RunWorkers` in one goroutine and calls `repo.ClaimJobs` on the caller. Workers signal a free slot after each job. `ClaimJobs` is not a goroutine.
- **Tests:** `go test -count=1 -timeout 20s -v ./internal/worker` → both tests PASS. `go test -race -count=1 -timeout 30s ./internal/worker` → PASS. `TestPoolFunction_ClaimsWaitingJobs` creates 5 `WAITING` jobs, 2 workers, and all end `COMPLETED`.
- **Next task:** Slice 4 — cancel the pool between jobs on `SIGINT`/`SIGTERM`.

---

### 2026-11-12 — M2 slice 4 cancel in the pool
- **Work:** `PoolFunction` checks `ctx` before every claim. The free-slot send in `RunWorkers` gives up on `ctx.Done()`. Terminal `UpdateState` uses `context.WithoutCancel` so the in-flight job can still be saved.
- **Tests:** `TestPoolFunction_CancelBeforeClaim` and `TestPoolFunction_CancelBetweenJobs` PASS (`go test -count=1 -timeout 20s ./internal/worker`).
- **Next task:** `main` cancels that same context on `SIGINT`/`SIGTERM`. Postgres demo stays optional.

---

### 2026-11-14 — Docs caught up; signal step clarified
- **Clarification:** `main` does not call `ClaimJobs`. It creates `WAITING` jobs, builds a context with `signal.NotifyContext` for `SIGINT` and `SIGTERM`, and passes that context to `PoolFunction`. `PoolFunction` is the only claimer. Cancelling the context stops new claims; it is not itself a claim.
- **Repo:** `internal/worker` (`RunWorkers`, `PoolFunction`) exists. `main.go` already has `signal.NotifyContext` and still only runs the Postgres CRUD demo.
- **Docs corrected:** `next-session.md`, `llm-tutor-brief.md`, `milestones.md` (two M2 boxes checked; gate still open), `README.md`. `teaching-notes.md` now requires jargon plus a bracketed explanation in every learner-facing reply.
- **Next task (learner):** wire `main` to `PoolFunction` with that signal context. Postgres demo stays optional.

---

### 2026-11-14 — M2 signal context in main
- **Work:** `main` creates `WAITING` jobs, passes `handler` (returns nil) into `worker.PoolFunction` with the `signal.NotifyContext` context, and does not call `ClaimJobs`.
- **Observed:** a `go run` at 23:21 created three `test-job-context` rows and Postgres showed them `COMPLETED` within the same second. `CreateJob`'s returned struct still prints `WAITING` because that value is the insert snapshot. `FindRunByID` after the pool still shows the run's own status; the pool does not call `UpdateRunStatus`.
- **Not observed this session:** Ctrl+C mid-run. Pool cancel tests from 2026-11-12 still cover cancellation inside `PoolFunction`.
- **Next task:** per-task deadlines. One job's time limit must end that job without cancelling the other workers. Process `SIGINT` stays the pool-wide stop.

---

### 2026-11-15 — M2 per-task deadline
- **Work:** `Handler` takes a `context`. `RunWorkers` and `PoolFunction` take a `time.Duration`. Each job gets `context.WithTimeout`. A handler that returns `ErrNetworkTimeout` is stored as `STALLED`.
- **Tests:** `TestRunWorkers_DeadlineStallsSlowJob` PASS (`go test -count=1 -timeout 20s -v ./internal/worker`). Slow job `STALLED`, fast job `COMPLETED`. The other four worker tests still PASS.
- **Observed:** `main` with work `5s` and deadline `3s` retried until Ctrl+C. `ClaimJobs` selects `STALLED`, and nothing caps the attempt count, so the pool never goes idle.
- **Next task:** attempt tracking and a max attempt count so a repeated `STALLED` job becomes `FAILED` and the pool can return. Execution duration is part of the same slice. Backoff and leases stay M3.

### 2026-11-15 — Persistent agent memory
- **Work:** Read active project documentation and inspected domain, repositories, worker/pool, main, migrations, and existing test cases. Added `docs/agent-memory.md` as a compact resume guide and root `AGENTS.md` to require reading and maintaining it after meaningful progress.
- **Findings:** Recorded stale bootstrap/tutor summaries and implementation gaps separately from intended architecture, including unlimited retries, missing leases/integration tests, Postgres state-validation/status-read gaps, and shared memory pointers.
- **Verification:** Documentation/code inspection only; no Go tests, build, application, or database commands run. Existing uncommitted work preserved; no implementation changes.
- **Next task:** Unchanged: learner-owned attempt count, latest duration, and maximum-attempt failure in M2, followed by CLI inspection and remaining shutdown evidence.

### 2026-11-16 — Persist per-job retry policy through claims
- **Work:** Per user request, continued the in-progress model changes through the repositories. `CreateJob` now accepts a per-job retry limit; memory claims increment attempt count while moving to `RUNNING`; PostgreSQL performs both changes in one `UPDATE ... RETURNING` statement and returns retry limit/count. Job reads scan these values. Updated call sites and tests; the claim test covers distinct limits and incrementing again after `STALLED`.
- **Scope:** Worker outcome handling (the user's next step 4) was left untouched. Latest attempt duration remains unfinished. No PostgreSQL integration test exists.
- **Verification:** `gofmt` ran on touched Go files. Tests and build were not run.
- **Next:** Review the repository changes and claim SQL, then implement worker validation/outcome policy.

### 2026-11-16 — Incremental retry metadata migration
- **Work:** Corrected the new `00002_update_to_db.sql` to add retry limit, attempt count, and latest-attempt-duration columns to the existing `jobs` table. Added a matching down migration. Left the older migration untouched.
- **Verification:** Inspected migration text only; did not apply it or run tests/build.
- **Note:** In `worker.go`, `job` names the imported package. `RetryLimit` is a field on a particular `Job`, so it must be selected through the job value (currently named `current` in that loop), not through the package name.

### 2026-11-16 — Record per-attempt execution duration
- **Work:** Added timing around each handler call. The worker now persists the resulting state and elapsed duration together through `UpdateAttemptOutcome`; memory and PostgreSQL repositories store it, and Postgres reads return the duration in nanoseconds. Corrected the migration column to BIGINT nanoseconds. The worker also rejects nil/non-RUNNING inputs before handler execution and reports the issue while releasing the pool slot. Added a focused duration test and updated the existing invalid-state expectation.
- **Scope:** No LLM diagnosis or raw failure-detail persistence added. Retry exhaustion behavior remains for the learner's next worker step.
- **Verification:** `gofmt` ran on touched Go files. Tests/build/migration were not run/applied.

### 2026-11-16 — Retry-limit semantics
- **Decision:** Per-job `RetryLimit` counts retries after the initial attempt. A value of 5 allows at most 6 total attempts. Success stays `COMPLETED`; permanent errors fail immediately; temporary errors stall while retries remain and fail after the final attempt.
- **Next:** Implement and test the retry-exhaustion decision in `RunWorkers`.

### 2026-11-16 — Add worker branch tests before retry implementation
- **Work:** Added tests for first/final-attempt success, permanent failure, temporary retry while attempts remain, temporary failure after retry exhaustion, zero retries, nil and non-running jobs, outcome-persistence failure, per-attempt duration, and cancellation while signaling a free slot. Added `t.Log` progress messages to the claim and worker tests for verbose runs. The retry-exhaustion case captures the agreed behavior and is expected to fail until `RunWorkers` implements it.
- **Verification:** Ran `gofmt` on `worker_test.go` and `git diff --check` for that file. Did not run tests or build, per the user's plan to run them.
- **Next:** User runs the worker tests; if failures match the unimplemented retry-exhaustion behavior, implement that outcome rule in `RunWorkers`.

### 2026-11-16 — Enforce retry exhaustion in worker
- **Evidence:** User ran the worker suite; all shown tests passed except `temporary_error_exhausts_retries`, which observed `STALLED` instead of `FAILED` on attempt 6 with retry limit 5.
- **Change:** `RunWorkers` now keeps success as `COMPLETED`, fails permanent errors immediately, fails temporary errors when `AttemptCount > RetryLimit`, and otherwise leaves them `STALLED` for another retry.
- **Verification:** Formatted `worker.go` and checked its diff. The worker tests have not been rerun after this change.
- **Next:** User reruns the worker tests and reports the result; then proceed if the retry matrix passes.

### 2026-11-16 — Retry slice user-confirmed; move to CLI inspection
- **User report:** “Looks good now” after the retry exhaustion change. The specific test command/output was not included, so this records user confirmation, not an independently verified pass.
- **Next:** M2 CLI job inspection, followed by evidence for graceful shutdown leaving work recoverable. Retry limit 5 means one initial attempt plus five retries.

