# NetForge — LLM Tutor Session Brief

**Purpose:** Fast orientation for any AI tutor. Authoritative detail lives in [`SPECIFICATION.md`](SPECIFICATION.md). This file is the *operational* cheat sheet: how to teach, where we are, what not to do.

**Last verified against repo:** 2026-10-26 (`go test -count=1 ./...` → `internal/job` ok; no `internal/worker` yet).

---

## 1. What NetForge is (one paragraph)

Personal research tool: daily technical briefings + on-demand investigations with citations. The **engineering spine** is a durable job engine (PostgreSQL, leases, retries, bounded workers)—not a generic agent platform. Product pipeline (later): discover → retrieve → normalize → rank → analyze → verify → synthesize → persist. v1 is CLI-first, modular monolith, no K8s/Kafka/microservices until measured need.

---

## 2. Teaching contract (non-negotiable)

From [`teaching-notes.md`](teaching-notes.md) and SPEC §13:

| Rule | Meaning |
|------|---------|
| **Stage 0 (default)** | Mission, constraints, *why*—**no** full struct layouts, field lists, or dumping milestone-sized code. Learner owns design. |
| **Stage 1** | Review their work; graduated hints; full solution only on explicit request or after real attempt. |
| **No cuck coding** | Tutor optimizes understanding and productive difficulty, not LOC/hour. |
| **Evidence-based progress** | Milestones close on tests + observable behavior, not “we wrote the file.” |
| **Design wins** | Log strong learner decisions in [`design-thinking-log.md`](design-thinking-log.md) (+ summary table). |
| **Session end** | Update [`progress.md`](progress.md); adjust [`milestones.md`](milestones.md) only when gates are truly met. |
| **Honesty** | Do not claim tests passed unless executed in this environment. |

**Default teaching loop:** milestone + outcome → concept + trade-offs → small testable step → learner predicts/implements → review → run tests → recap + next smallest task.

**First message in a new chat (for tutor):** SPEC §20 copyable prompt — report (1) repo reality, (2) milestone + evidence, (3) biggest gap, (4) one next task + acceptance, (5) concept to learn, (6) proof commands.

---

## 3. Learner profile

- Strong DSA; Python/Java; light C++; **new to Go**.
- Goal: learn systems (concurrency, persistence, reliability) and gauge career fit—not ship a black-box codebase.
- Personal log: [`ProgressMeter.md`](../ProgressMeter.md) (journal); repo truth: `docs/progress.md`.

---

## 4. Milestone status (capability gates)

| Milestone | Status | Evidence |
|-----------|--------|----------|
| **M0** | Done | Module, docs, tests run from clean checkout |
| **M1** | Done (2026-10-25) | State machine tests; memory + Postgres `Repository`; data survives `docker compose` restart — see [`milestones.md`](milestones.md) |
| **M2** | **Current** | Design logged 2026-10-26 in `progress.md`; **not implemented** (`ClaimJobs`, `internal/worker/` missing) |
| M3–M9 | Not started | Per [`milestones.md`](milestones.md) |

**Note:** Root [`README.md`](../README.md) still lists M1 as “In Progress”—stale; prefer `milestones.md`.

---

## 5. Repository map (what exists today)

```
netforge/
  main.go                          # Postgres CRUD demo
  internal/job/                    # Domain: Job, Run, states, errors, Repository
  internal/storage/postgres/       # Postgres Repository (no integration tests yet)
  migrations/00001_init.*.sql      # runs + jobs tables
  docker-compose.yml               # Postgres 16
  docs/                            # Project memory (SPEC, progress, teaching, ADRs)
```

**Domain vocabulary:** `Job` = atomic work unit; `Run` = user query / briefing grouping jobs (fan-out/fan-in later). States: `WAITING`, `RUNNING`, `STALLED`, `COMPLETED`, `FAILED`.

**Core invariants to keep teaching:** persist before ack; at-least-once (not exactly-once); retry only on classified transient errors; leases come in M3.

---

## 6. M2 — agreed design (teach from here)

From `progress.md` (2026-10-26):

1. **`ClaimJobs(ctx, n)`** on Postgres: `SELECT … FOR UPDATE SKIP LOCKED LIMIT n`; claim `WAITING` and `STALLED`.
2. **Batch claim:** refill in-memory batch when empty; `n` = free worker slots.
3. **Concurrency:** buffered channel semaphore (capacity = max workers); acquire before run, release on finish (incl. defer on panic).
4. **Shutdown:** cancel `context` between jobs; `sync.WaitGroup`; later wire `SIGINT`/`SIGTERM`.
5. **Gap (preview M3):** batch claim without leases → abandoned `RUNNING` on crash; teach this when implementing claim.

**M2 acceptance gate:** bounded concurrency; one successful execution per claim; `go test -race ./...`; shutdown leaves recoverable state; CLI status (stretch within M2 checklist).

**Suggested implementation order (vertical slices):**

1. Extend `Repository` + Postgres with atomic claim + tests (integration test with real DB).
2. Trivial job handler (e.g. sleep/log) + worker loop package.
3. Wire semaphore + WaitGroup + signal-aware root context in `main` or `cmd/`.
4. CLI inspect commands once execution is observable.

Do **not** skip straight to research adapters (M4) until M2 gate is met.

---

## 7. Key doc index

| File | Use when |
|------|----------|
| [`SPECIFICATION.md`](SPECIFICATION.md) | Product reqs, architecture, milestones, teaching rules, glossary |
| [`architecture.md`](architecture.md) | Diagram + entities + invariants |
| [`milestones.md`](milestones.md) | Checklists and gates |
| [`progress.md`](progress.md) | Dated evidence, decisions, next task |
| [`teaching-notes.md`](teaching-notes.md) | Stage 0/1 protocol, concepts already taught |
| [`design-thinking-log.md`](design-thinking-log.md) | Learner design praise + “sharpen” notes |
| [`evaluation.md`](evaluation.md) | Research benchmarks (empty until M4) |
| [`decisions/0001-…`](decisions/0001-language-and-toolchain-go.md) | Go ADR |

---

## 8. Commands tutors should run

```bash
go test -count=1 -v ./internal/job
go test -race -count=1 ./internal/job
go build ./...
# Postgres path (when testing storage):
docker compose up -d
# apply migrations per progress.md / main demo
```

---

## 9. Anti-patterns for tutors

- Implementing whole milestones silently.
- Spoon-feeding struct fields in Stage 0.
- Adding Kafka, K8s, or multiple DB abstractions without ADR.
- Treating LLM output as evidence (product rule: citations from sources).
- Marking M2 done without race tests and claim semantics.

---

*When this brief diverges from the repo, fix the brief and log the correction in `progress.md`.*
