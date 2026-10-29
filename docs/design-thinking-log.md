# NetForge — Design Thinking Log

A record of the learner's original design ideas, what was strong about each, and what to sharpen going forward.

---

## Job State Machine

### Your idea
Five states: `WAITING → RUNNING → COMPLETED / FAILED`, with `STALLED` as an intermediate recoverable state for transient failures.

### What was good
- Correctly separated terminal states (`COMPLETED`, `FAILED`) from recoverable ones (`STALLED`).
- Recognised that a job can re-enter `RUNNING` from `STALLED` — not all failures are permanent.
- Mapped cleanly onto a real production state machine (very close to what systems like Celery and Temporal use).

### What to sharpen
- The transition from `STALLED → RUNNING` needs a retry limit — otherwise a permanently broken job bounces forever. You'll add `AttemptCount` and a max-retry policy in M3.
- Consider who is responsible for moving a job to `STALLED` vs `FAILED` — that logic belongs in the error classifier, not scattered across the worker.

---

## Error Classification (`ErrorCategory`)

### Your idea
Separate the *what happened* (the `error` value) from *what to do about it* (`ErrorCategory`: retry vs no-retry).

### What was good
- Clean separation of concerns — the worker doesn't need to understand error details, just the category.
- Directly enables bounded retry logic in M3 without changing the worker loop.
- Mirrors how production systems classify errors (transient network failure = retry; invalid payload = permanent failure).

### What to sharpen
- The current classifier is a simple `errors.Is` check. Real classifiers also inspect HTTP status codes, timeout signals, and domain-specific sentinel errors. You'll extend this in M3.

---

## Repository Interface + Dependency Injection

### Your idea
Define a `Repository` interface in the domain package, implement it twice (in-memory for tests, PostgreSQL for production), and inject whichever one is needed.

### What was good
- Correct understanding that interfaces define *behaviour contracts*, not implementations.
- Immediately made unit tests possible without a running database.
- Allowed the PostgreSQL implementation to be swapped in without changing any domain logic.

### What to sharpen
- The interface currently mixes Job and Run operations in a single type. As the system grows, you may want to split these into `JobRepository` and `RunRepository` to keep each interface small and focused (Interface Segregation).

---

## Batch Job Claiming

### Your idea
Instead of one DB call per worker per job, fetch `N` jobs at once (`LIMIT free_workers`) and distribute them from an in-memory list. Re-query only when the list is empty.

### What was good
- Correctly identified that N+1 DB queries is wasteful when you already know how many workers are free.
- The `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N` pattern is exactly how production queues do batch claiming.
- Including `STALLED` alongside `WAITING` in the claim query is correct — stalled jobs eligible for retry should be re-claimable.

### What to sharpen
- If the process crashes after claiming a batch (all marked `RUNNING`) but before executing, those jobs are stuck forever. This is the abandoned-job problem — the fix is leases (`lease_expires_at`) which you'll add in M3.
- The batch size `N` should be the number of *currently free* workers, not the total pool size — which leads naturally to the worker registry idea below.

---

## Worker Registry (In-Process State Tracker)

### Your idea
Maintain a small in-process data structure keyed by worker ID that tracks each worker's current status (free / busy). Use this to compute `N` (free workers) before issuing a batch claim query, avoiding a DB round-trip for worker state.

### What was good
- Correct instinct that worker state is ephemeral and doesn't belong in the DB — it's process-local.
- Enables efficient batch sizing without extra queries.
- Separates worker lifecycle management from job execution logic cleanly.

### What to sharpen
- Multiple goroutines will update this structure concurrently — it needs synchronization. Two clean options in Go: a `sync.Mutex` wrapping a map, or a single goroutine that owns the map and communicates via channels. Think about which fits the worker loop design better.
- Worker IDs need to be stable within a run — a simple integer index (`0..N-1`) works fine for a single-process pool.

---

## Graceful Shutdown

### Your idea
Make the main goroutine wait for all worker goroutines to finish using `sync.WaitGroup` before exiting.

### What was good
- Correct — without `WaitGroup`, the process exits immediately and kills mid-flight workers.
- `context.Context` cancellation + `WaitGroup` is the standard Go pattern for this.

### What to sharpen
- `WaitGroup.Wait()` alone doesn't guarantee workers finish *cleanly* — a worker mid-execution when `ctx` is cancelled should complete its current job before exiting (drain, don't drop). Make sure the worker loop checks `ctx.Done()` only between jobs, not mid-execution.
- In production you'd also handle `SIGINT`/`SIGTERM` signals to trigger the cancellation. You'll add this in M2.

---

## Summary Table

| Idea | Strength | Main gap to address |
|---|---|---|
| Five-state machine with STALLED | Recoverable vs terminal distinction | Add retry limit (M3) |
| Error classification | Decouples policy from mechanism | Extend to HTTP/timeout errors (M3) |
| Repository interface + DI | Testability, swappability | Consider splitting Job/Run repos later |
| Batch claim with SKIP LOCKED | Efficient, correct isolation | Abandoned-job risk → leases (M3) |
| Worker registry | No DB round-trips for worker state | Needs mutex or channel for thread safety |
| WaitGroup shutdown | Correct pattern | Drain in-flight jobs before exit |

---

## Bounded Concurrency Mechanism

### Your idea (chosen approach)
After considering both options — a worker registry (map + mutex) vs a buffered channel as semaphore — you chose the **channel approach**: channel capacity = max workers; each worker acquires a slot before starting a job and releases it on completion.

### What was good
- Recognised that the channel itself *is* the concurrency tracker — no separate data structure, no mutex, no dispatcher goroutine needed.
- This is idiomatic Go: channels communicate ownership and signal availability simultaneously.
- Simpler to reason about under failure conditions — a panicking worker that defers its channel release still works correctly.

### What to sharpen
- A buffered channel semaphore controls *how many run at once* but doesn't assign specific jobs to specific workers. The distribution still needs a work-source — a shared job queue channel or a dispatcher that feeds jobs in. Think about where jobs come from once a slot opens up.


