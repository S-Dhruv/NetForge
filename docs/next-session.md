# NetForge — Pick up here (session handoff)

**Purpose:** Single place to resume work in a new chat. Read this first, then [`progress.md`](progress.md) and [`teaching-notes.md`](teaching-notes.md).

**Last updated:** 2026-11-16 (retry exhaustion implemented; next is CLI job inspection)

---

## For the learner (Dhruv)

**Milestone:** M2 — single-process worker pool (not done yet).

**Already in the repo (you can rely on this):**

- M1 complete: job state machine, memory + Postgres repositories, migrations, Docker Postgres.
- **`ClaimJobs`:** implemented on memory + Postgres; test `internal/job/claim_jobs_test.go`.
- **`internal/worker/`:** slices 2–4 are in. `RunWorkers` consumes a job channel. `PoolFunction` is the only caller of `ClaimJobs`. The pool stops claiming when its `context` is cancelled (`TestPoolFunction_CancelBeforeClaim`, `TestPoolFunction_CancelBetweenJobs`).
- **`main.go`:** builds a context with `signal.NotifyContext` for `SIGINT` and `SIGTERM`, creates `WAITING` jobs, and calls `worker.PoolFunction` with a 3s deadline. The handler waits 5s or returns `ErrNetworkTimeout` if the job context ends first. It does **not** call `ClaimJobs`. That 5s/3s pair currently retries `STALLED` jobs until Ctrl+C.

**Who calls what:**

- Workers never call `ClaimJobs`.
- Tests of the pool do not call `ClaimJobs`. They create `WAITING` jobs and call `PoolFunction`.
- **`main` also does not call `ClaimJobs`.** It calls `PoolFunction`. `PoolFunction` claims.
- `SIGINT` / `SIGTERM` do not claim anything. They cancel the context you pass into `PoolFunction`. A cancelled context means "stop taking new work."

**Agreed runtime shape (your design):**

- **Central claimer** inside `PoolFunction` is the **only** caller of `ClaimJobs`.
- **`n`** = number of free worker slots, not always max pool size.
- Claimed jobs go on a **`jobs` channel**; worker goroutines read from it and execute.
- When a worker finishes, it signals the claimer on `free` so it can claim more.
- On finish: **`UpdateState`** on the repo (not `TransitionTo` from the worker, not `UpdateRunStatus` — that’s for runs).
- Terminal states: success → `COMPLETED`; handler error + temporary classifier → `STALLED`; else → `FAILED`.
- Jobs must be **`RUNNING`** before those transitions (from `ClaimJobs` or explicit transition in tests).
- Empty queue: if `ClaimJobs` returns nothing and every worker is idle, close the job channel and return. If some workers are still busy, wait on `free` or `ctx.Done()`.

**Current slice — CLI job inspection:**

**Goal:** Add a small CLI path to inspect a job's persisted state and retry progress. Keep M2 shutdown evidence open until a live signal run confirms accepted work is left in a recoverable state.

**Current implementation:** Job attempts are counted at claim time. The worker records latest attempt duration and chooses `FAILED` for a temporary error once `AttemptCount > RetryLimit`; success is `COMPLETED`, permanent errors fail immediately. `RetryLimit=5` means six total executions.

**Do not:** add backoff, jitter, or leases. Those are M3.

### After this slice (still M2, later)

- CLI inspect commands.

### After M2 gate (later)

- M3: leases, backoff, abandoned-job recovery. A crash today can leave a job `RUNNING` with no lease. That gap is known and out of scope for the attempt-count step.

**Verify anytime:**

```bash
go test -count=1 -v ./internal/job
go test -count=1 -timeout 20s -v ./internal/worker
go test -race -count=1 -timeout 30s ./internal/worker
```

---

## For any AI tutor in a new session

**Read before suggesting code:** [`teaching-notes.md`](teaching-notes.md) (Stage 0 / Stage 1, jargon rule), [`SPECIFICATION.md`](SPECIFICATION.md) §13.

**Non-negotiable:**

- **Learner implements.** Do not create or overwrite `.go` files unless the user explicitly asks (e.g. “apply this patch” / “fix this compile error in my file”).
- **Stage 0 default:** goals, acceptance criteria, concepts, one small next step — **no** full implementations, **no** spoon-fed struct field lists.
- **Stage 1:** review learner’s code; hints in steps; full solution only on **explicit** request.
- **Jargon:** every reply to the learner uses the technical term, then the meaning in brackets in the same sentence.
- Do **not** claim tests passed unless you ran them in that session.
- Do **not** tell the learner to call `ClaimJobs` from `main`. The signal cancels the context passed to `PoolFunction`.
- Log learner design wins in [`design-thinking-log.md`](design-thinking-log.md); append evidence to [`progress.md`](progress.md) after real work.

**First reply in a new chat should report:** what exists in repo, current slice (attempt count and duration), acceptance for next step, one concept to understand, commands to prove done — not a code dump.

**Stale doc:** if [`llm-tutor-brief.md`](llm-tutor-brief.md) lags, this file + `progress.md` override for “what’s next.”

---

## References

- M2 design notes: `progress.md` § 2026-10-26 — M2 Design, and 2026-11-12 slices
- Architecture: `architecture.md`, `milestones.md` (M2 checklist)
- Tutor entry: `llm-tutor-brief.md`
