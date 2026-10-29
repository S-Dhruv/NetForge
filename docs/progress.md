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

