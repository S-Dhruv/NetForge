# NetForge — Pick up here (session handoff)

**Purpose:** Single place to resume work in a new chat. Read this first, then [`progress.md`](progress.md) and [`teaching-notes.md`](teaching-notes.md).

**Last updated:** 2026-10-30

---

## For the learner (Dhruv)

**Milestone:** M2 — single-process worker pool (not done yet).

**Already in the repo (you can rely on this):**

- M1 complete: job state machine, memory + Postgres repositories, migrations, Docker Postgres.
- **`ClaimJobs`:** implemented on memory + Postgres; test `internal/job/claim_jobs_test.go`.
- **`internal/worker/`:** does **not** exist yet — you build it.

**Agreed runtime shape (your design):**

- **Central claimer** (one goroutine) is the **only** caller of `ClaimJobs`.
- **`n`** = number of free worker slots, not always max pool size.
- Claimed jobs go on a **`jobs` channel**; worker goroutines read from it and execute.
- When a worker finishes, signal the claimer so it can claim more (e.g. `workerReady` channel or equivalent).
- On finish: **`UpdateState`** on the repo (not `TransitionTo` from the worker, not `UpdateRunStatus` — that’s for runs).
- Terminal states: success → `COMPLETED`; handler error + temporary classifier → `STALLED`; else → `FAILED`.
- Jobs must be **`RUNNING`** before those transitions (from `ClaimJobs` or explicit transition in tests).

**Implement in this order (vertical slices):**

### Slice 2 — Workers only (no claimer yet)

**Goal:** Given jobs on a channel, N goroutines run a handler and persist terminal state.

**Build:**

1. `internal/worker` package: a **Handler** (work function) and a narrow **store** interface with `UpdateState` only (memory repo satisfies it).
2. One goroutine: loop reading from channel until **closed**; per job: handler → classify error → `UpdateState`.
3. **RunWorkers** (or similar): start **N** such goroutines; **WaitGroup**; caller **closes** channel when done sending.

**Test (you write):** memory repo, create jobs, set to **RUNNING**, send on channel, close, run 2 workers on 3 jobs, assert all **COMPLETED** with a stub handler that returns no error.

**Proof:** `go test -count=1 -v ./internal/worker`

**Do not:** call `ClaimJobs` inside workers in this slice.

---

### Slice 3 — Central claimer

**Goal:** Claimer fills the channel; workers only consume.

**Build:**

1. Claimer loop: when a worker is free, `ClaimJobs(ctx, k)` and push each job on `jobs`.
2. Workers: after each job, notify claimer (free slot).
3. Handle **empty queue** (no claimable rows): avoid deadlock — workers blocked on `<-jobs` must not stall forever; claimer retries with backoff or only signals when appropriate (document your choice in [`design-thinking-log.md`](design-thinking-log.md)).

**Test:** create **WAITING** jobs only; start pool; all end **COMPLETED** without the test calling `ClaimJobs` directly.

**Proof:** `go test -race -count=1 ./internal/worker`

---

### Slice 4 — Shutdown and demo

**Goal:** Clean exit; optional Postgres demo in `main`.

**Build:**

1. `Run(ctx)`: cancel context stops workers **between** jobs; WaitGroup waits for in-flight work.
2. `SIGINT` / `SIGTERM` in `main` cancels ctx (not only a timeout).
3. Optional: `main` creates run + jobs in **WAITING**, runs pool against Postgres.

**Note:** jobs left **RUNNING** after crash → M3 leases; don’t scope-creep into M3 for M2 gate.

---

### After M2 gate (later)

- CLI inspect commands (milestone checklist).
- M3: leases, backoff, abandoned-job recovery.

**Verify anytime:**

```bash
go test -count=1 -v ./internal/job
go test -count=1 -v ./internal/worker   # once package exists
go test -race -count=1 ./...
```

---

## For any AI tutor in a new session

**Read before suggesting code:** [`teaching-notes.md`](teaching-notes.md) (Stage 0 / Stage 1), [`SPECIFICATION.md`](SPECIFICATION.md) §13.

**Non-negotiable:**

- **Learner implements.** Do not create or overwrite `.go` files unless the user explicitly asks (e.g. “apply this patch” / “fix this compile error in my file”).
- **Stage 0 default:** goals, acceptance criteria, concepts, one small next step — **no** full implementations, **no** spoon-fed struct field lists.
- **Stage 1:** review learner’s code; hints in steps; full solution only on **explicit** request.
- Do **not** claim tests passed unless you ran them in that session.
- Log learner design wins in [`design-thinking-log.md`](design-thinking-log.md); append evidence to [`progress.md`](progress.md) after real work.

**First reply in a new chat should report:** what exists in repo, current slice (2), acceptance for next step, one concept to understand, commands to prove done — not a code dump.

**Stale doc:** [`llm-tutor-brief.md`](llm-tutor-brief.md) may lag; this file + `progress.md` override for “what’s next.”

---

## Open questions (learner to decide when implementing slice 3)

- Exact signal from worker → claimer when a slot frees.
- Behavior when `ClaimJobs` returns zero (backoff vs blocking strategy).
- Whether pool size alone bounds concurrency or you also use a semaphore channel (design log discussed both; channel-only pool size is enough for M2 if workers == max concurrent jobs).

---

## References

- M2 design notes: `progress.md` § 2026-10-26 — M2 Design  
- Architecture: `architecture.md`, `milestones.md` (M2 checklist)  
- Tutor entry: `llm-tutor-brief.md`
