# NetForge — persistent agent memory

Last reviewed: 2026-11-16. Based on the current working tree, including existing
uncommitted changes. This is a compact resume guide; code determines implemented
behavior, and the specification defines intended behavior. Verify changed files
instead of rereading the whole documentation set each session.

## Purpose and working contract

NetForge is Dhruv's personal research companion: on-demand investigations and
daily technical briefings with traceable citations. The foundation is a durable
job engine built in Go. Dhruv is learning systems engineering to explore career
fit; strong DSA, Python/Java experience, light C++, new to Go.

- Learner owns implementation. Do not write/overwrite `.go` files without an
  explicit request. Documentation maintenance is authorized.
- Stage 0 default: explain goal, constraints, acceptance, concept, and one small
  task; no code snippets or prescribed fields/schema. Stage 1: review their
  attempt and give graduated hints; full solutions only on explicit request.
- Acknowledge correct reasoning before corrections. Explain unfamiliar technical
  terms immediately in brackets. Own incorrect advice plainly.
- Progress requires observed evidence. Separate historical test results, current
  inspection, and commands actually run this session.
- Prefer standard library, small changes, CLI first, modular monolith, PostgreSQL.
  Defer microservices, Kafka, Kubernetes, extra databases, and speculative layers.
- Target at-least-once execution with idempotency; never promise exactly-once.
  Persist accepted work; eventually enforce leases, bounded retries, and recovery.
- Later research must preserve provenance, verify citation support, label
  uncertainty, bound costs, and treat retrieved content as untrusted data.

## Current checkpoint

M0 and M1 recorded complete; M1 restart durability was observed on 2026-10-25.
**M2 is in progress.** M3–M9 have not started.

Implemented: job lifecycle, memory/Postgres storage, atomic claims, bounded
workers, pool cancellation, signal context in main, per-job timeout contexts.
Retry tracking is in progress: per-job retry limit and attempt count are in the
model/schema, claims increment the count, and the worker records each attempt's
elapsed duration with its outcome. Worker tests now cover outcome branches,
invalid inputs, persistence errors, and cancellation signaling. The worker now
fails temporary errors after `AttemptCount > RetryLimit`. User said the result
looks good after the rerun, but did not include command/output here; treat that
as user-confirmed, not independently verified. Pending: CLI inspection and full
shutdown evidence.

**Next learner task:** CLI status inspection for jobs, including state and attempt
information. M2 shutdown evidence remains open. `RetryLimit` means additional
retries, so limit 5 allows six total attempts. Backoff, jitter, leases, and
abandoned-job recovery remain M3.

## Code map and actual behavior

| Path | Responsibility / important detail |
| --- | --- |
| `go.mod` | Module `netforge`, declared Go `1.27.2`, pgx/v5 `v5.11.0`; installed toolchain not rechecked for this documentation task. |
| `internal/job/job.go`, `state.go` | Job data includes retry limit, attempt count, and latest attempt duration. Duration is Go `time.Duration`, stored in PostgreSQL as nanoseconds. Transitions: `WAITING → RUNNING`; `RUNNING → STALLED/FAILED/COMPLETED`; `STALLED → RUNNING`; terminal states have no next transition. |
| `internal/job/run.go` | `RunTask` groups jobs; `InitializeRun` validates query, defaults max jobs to 6. Dependency orchestration is future work. |
| `internal/job/errors.go` | nil succeeds; `ErrNetworkTimeout` and `JobCreationError` are temporary; all other errors are permanent. Raw context errors are not specially classified. |
| `internal/job/repository.go` | Broad Job + Run storage interface. Memory repository currently implements only the job operations used by tests/workers, not the full interface. |
| `internal/job/memory_repository.go` | Mutex-protected job map; state writes use `TransitionTo`; reads return stored pointers. Claims sort eligible jobs by ID, mark them `RUNNING`, and increment attempt count under the lock. |
| `internal/storage/postgres/postgres_repository.go` | `database/sql` repository; claims use one `UPDATE … SELECT … FOR UPDATE SKIP LOCKED … RETURNING` statement to mark `RUNNING`, increment attempt count, and return per-job retry limit/count. Eligible states `WAITING`/`STALLED`, ordered by ID, not stalled-first. |
| `internal/worker/worker.go` | `Handler(ctx, *Job) error`; workers reject nil/non-`RUNNING` jobs before calling the handler. `RunWorkers` times each attempt and stores its state plus duration, then signals `free`. Collects write/input errors with a mutex and joins them after waiting. Retry exhaustion is unfinished. |
| `internal/worker/claimer.go` | `PoolFunction` alone calls `ClaimJobs`; workers and main never claim directly. Owns unbuffered `jobs`, buffered `free`, and one goroutine running `RunWorkers`. Claims up to available slots; returns when no eligible jobs and no in-flight jobs remain. |
| `main.go` | Postgres demo using `DATABASE_URL`; signal context for SIGINT/SIGTERM; creates three runs/jobs, starts 3 workers with 3s timeout. Handler waits 5s or returns `ErrNetworkTimeout` on cancellation. Does not aggregate run status. |
| `migrations/00001_init.*.sql`, `00002_update_to_db.sql` | Initial schema plus an incremental migration for retry limit, attempt count, and latest-attempt-duration columns. A down migration reverses those columns. |
| `docker-compose.yml` | PostgreSQL 16 Alpine, host port 5432, named persistent volume. `.env` is private; never copy its contents into memory or logs. |

Cancellation stops new claims/sends and closes the jobs channel, then waits for
workers. Handlers must cooperate with their context: timeout does not forcibly
interrupt arbitrary Go code. Final state writes use `context.WithoutCancel(ctx)`.
Worker success becomes `COMPLETED`, temporary failure `STALLED`, permanent failure
`FAILED`. The pool returns persistence errors; handler errors become job states.

## Known gaps and documentation traps

- Main's 5s work / 3s timeout repeatedly produces `STALLED`; immediate reclaim
  loops until cancellation because there is no maximum attempt count.
- A crash or cancellation after claiming but before dispatch can leave rows
  `RUNNING`. No leases/recovery yet; do not claim full shutdown safety.
- Postgres `UpdateState` does not validate domain transitions or missing IDs,
  despite its comment; memory storage does validate transitions.
- Postgres `FindRunByID` scans status into a string but does not assign `Status`
  on the returned struct. Separately, workers never update run status.
- Memory storage returns shared pointers; its mutex does not protect arbitrary
  caller access after return. Passing existing race tests is not a general proof.
- No Postgres integration tests exist. Historical restart checks are manual.
- `architecture.md` lists future metadata/invariants, not all implemented ones.
- `llm-tutor-brief.md` has stale main/signal/slice descriptions; older design
  notes mention a semaphore and draining-only cancellation. Current code uses
  fixed consumers plus free-slot signals and cooperative handler cancellation.
- `netforge_project_context/` is an old bootstrap context pack with obsolete
  planning checkpoints; use active root `docs/` for current work.

These are inspection findings, not changes requested or implemented this session.

## Verification and evidence

Current tests cover transitions; memory CRUD/concurrency/claiming; five mixed
worker outcomes; pool claims; cancellation before claim and between jobs; slow
job deadline with fast job completion. Storage package has no tests.

Historical evidence: worker tests and race tests passed 2026-11-12; deadline test
and other four worker tests passed 2026-11-15 according to `progress.md`.
On 2026-11-16, repository claim behavior and tests were edited, then gofmt ran;
no tests, build, database, or application were run for those edits. `00002_update_to_db.sql`
adds metadata to an existing jobs table and stores elapsed duration as nanoseconds;
the migration has not been applied.

Use relevant checks after implementation (quote `./...` to avoid zsh autocorrect):

```sh
go test -count=1 -timeout 20s -v ./internal/worker
go test -race -count=1 -timeout 30s ./internal/worker
go test -count=1 -v ./internal/job
go test -race -count=1 './...'
go build './...'
go vet './...'
```

Do not run main casually to verify docs: it writes database rows and the current
demo retries indefinitely. Database integration requires a disposable database.

## Reading and update protocol

At resume: read this guide, inspect `git status --short` and relevant changed
code, then consult `next-session.md` if needed. Preserve pre-existing edits.
For current task conflicts, inspect code and the latest `progress.md` /
`next-session.md`; specification remains the product/teaching contract.

After meaningful progress, update this guide's review date, checkpoint, actual
behavior, known gaps, verification, and next acceptance criteria. Replace stale
facts rather than growing a transcript. Also:

- Append dated evidence and next task to `progress.md`.
- Refresh `next-session.md` when the handoff changes.
- Update `milestones.md` only when acceptance evidence changes.
- Record learner decisions/sharpening in `design-thinking-log.md` and its table;
  teaching insights/preferences in `teaching-notes.md`.
- Update architecture and ADRs only for real design decisions; sync README and
  tutor brief when their summaries change. Never invent progress or decisions.

Read deeper only as needed: `SPECIFICATION.md` for requirements (§4), lifecycle
(§5), architecture (§6), research (§7), reliability/security (§8), tests (§9),
gates (§10), teaching (§13), done/memory (§14–15); `architecture.md` for intended
design; `milestones.md` for gates; `progress.md` for history; `teaching-notes.md`
for pedagogy; `design-thinking-log.md` for learner reasoning; `decisions/` for
ADRs; `evaluation.md` for future research evaluation. `Working.md` and
`ProgressMeter.md` are personal journals, not implementation evidence.
