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
| Narrow `Store` on the worker | Caller can only persist state | Channel still belongs to the caller, not the loop |
| Mixed worker test | Success, bad payload, and illegal state in one run | Failed writes are joined after `Wait`, not by stopping the loop |
| Wake the claimer when a worker finishes | Exact moment a slot is free | Initial `n` free slots, and an empty claim, still need their own rules |
| Claim in free-slot batches, stalled first | Reuse `ClaimJobs`; retry before new work | One push cannot be the whole claimer; current claim sorts by id |
| Gate sends on cancel | Stop new work between jobs | `ctx` is already that gate; a bool does not wake a parked send |
| Demo handler lives in `main` | Pool stays free of demo work; `ClaimJobs` stays inside `PoolFunction` | `CreateJob`'s return value is the insert snapshot; run status is a different row from job state |

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

---

## Narrow worker store

### Your idea
The worker package declares its own `Store` with only `UpdateState`, instead of taking the full `job.Repository`.

### What was good
- The worker cannot call `ClaimJobs` or run methods by accident. The compiler enforces the slice boundary.
- `MemoryRepository` already has `UpdateState`, so it satisfies `Store` with no wrapper and no second implementation.

### What to sharpen
- The jobs channel is created by the caller (the test, later the claimer). The worker function only receives it and ranges over it. Creating the channel inside the worker loop leaves nobody who can send.
- `Store` does not embed `Repository` or gain its other methods. It lists `UpdateState` only. A repository value is accepted because that one method matches.

---

## Mixed outcomes in one worker run

### Your idea
One test pushes five jobs: three normal, one with data the handler rejects, and one left in a state that cannot move to `COMPLETED`.

### What was good
- A single happy-path assertion would hide both the classifier and the state machine. The bad payload must end `FAILED`, and the `WAITING` job must stay `WAITING` because `UpdateState` goes through `TransitionTo`.
- Two workers and five jobs also forces the loop to run more than once per goroutine.

### What to sharpen
- A failed `UpdateState` must not `return` from the worker goroutine. That goroutine would stop ranging, and if every worker did the same the jobs still in the channel would never run. Record the error, `continue`, and return the joined errors only after `Wait`.

---

## Free-slot wake versus sleep

### Your idea
The worker that finishes a job wakes the claimer immediately. A sleep loop was the other option: the claimer naps and checks later whether a slot might be free.

### What was good
- The worker is the only place that knows the slot became free. Waking the claimer at that moment removes the guesswork from `time.Sleep`.
- A claimer blocked on a signal is already asleep. The worker's send is the wake-up. A separate sleeper goroutine would only repeat that signal.

### What to sharpen
- Free slot and empty queue are different waits. No claimable rows is not solved by sleeping until a worker is free. If every worker is idle and `ClaimJobs` returns nothing, the pool is finished. If some workers are still busy, block until one of them signals, then claim again.
- At startup every slot is free and no worker has finished yet. If the claimer waits for a wake-up before the first claim, and the workers wait for a job before they send that wake-up, both sides block. The claimer has to treat the initial `n` slots as free without a signal.

---

## Claimer loop, batch size, and stalled-first

### Your idea
One function calls the existing `ClaimJobs`, in batches sized from a local free-slot count that workers update. Prefer `STALLED` over `WAITING`. Create a channel, push the batch, then run the workers. A single function looked sufficient because the work is “query, then push.”

### What was good
- Reusing `ClaimJobs` instead of writing a second query. The repository already locks, selects `WAITING` and `STALLED`, and moves them to `RUNNING`.
- The free-slot count is process-local. Workers are the ones who know a slot opened.
- Claiming `STALLED` before `WAITING` retries work that already failed transiently before starting new work.

### What to sharpen
- The claimer function has to stay in the loop until `ClaimJobs` returns nothing and every worker is idle. A single push leaves later `WAITING` rows unclaimed, because the next claim happens only after a worker finishes.
- The claimer can be a normal function on the caller goroutine. The workers are already goroutines. Launching the claimer with `go` is only required if something else must run on the caller at the same time.
- Batch size is the number of free slots. Claiming extra rows marks them `RUNNING` while they sit in the channel. That cuts database calls and widens the crash window.
- Today `ClaimJobs` does not prefer `STALLED`. It collects both states and sorts by id. Stalled-first is a change inside that method. A steady stream of stalled jobs can starve `WAITING` ones.
- The jobs channel is created once. `RunWorkers` is what starts the `n` goroutines, and a direct call does not return until the channel is closed. The pool has to run that call concurrently with the claim loop. Claiming first, then calling `RunWorkers`, either claims every row before a worker runs or never reaches the second claim. An open empty channel blocks the workers; it does not make `RunWorkers` return.

---

## Cancel before the next send

### Your idea
Before anything else is sent on the job channel, check a condition, possibly a bool, and refuse further sends once the pool is stopping.

### What was good
- The right moment to stop is before the next claim and before the next send. The handler already running should finish.
- A gate in front of the channel keeps unclaimed rows in `WAITING`.

### What to sharpen
- `ctx.Done()` is already that switch. A separate bool duplicates it, races if another goroutine writes it, and can disagree with the context `main` will cancel.
- The bool does not wake a goroutine parked on a receive or on `free <-`. Shutdown still has to close the job channel, and the free-slot send has to be able to give up when the context is cancelled. Otherwise `Wait` never returns.

---

## Handler in main, passed as a value

### Your idea
A small function in `main.go` returns nil, and that function value is the third argument to `PoolFunction`.

### What was good
- The worker package stays unaware of the demo. `Handler` is a function type, so any matching function in `main` is valid.
- `main` still does not call `ClaimJobs`. The signal context and the pool call are the process's whole job.

### What to sharpen
- The struct `CreateJob` returns is the insert snapshot. Printing it after creation always shows `WAITING`, including when the row is later `COMPLETED`.
- `FindRunByID` reads the run row. The pool updates job rows through `UpdateState` and does not call `UpdateRunStatus`, so the run can stay `WAITING` after every job is `COMPLETED`.


