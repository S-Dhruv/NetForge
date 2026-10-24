# NetForge

A personal research companion powered by a durable, observable, and fault-tolerant job execution engine, built in Go.

---

## Capabilities & Roadmap
- **M0: Repository & Contract** — Go module, testing harness, project charter. *(Complete)*
- **M1: Domain Model & Durable Jobs** — State machine, Job/Run entities, PostgreSQL persistence. *(In Progress)*
- **M2: Single-Process Worker Pool** — Bounded concurrency, atomic job claiming, graceful shutdown.
- **M3: Retry & Crash Recovery** — Leases, exponential backoff, abandoned job recovery.
- **M4: First Useful Research Workflow** — Safe retrieval, normalization, and reporting.
- **M5: Evidence-Backed Analysis** — Structured claims and citation verification.
- **M6: On-Demand Research Orchestration** — Subtask planning and dependency graphs.
- **M7: Daily Technical Briefing** — Scheduled digests with novelty ranking.
- **M8: Personal Research Memory** — Topic history, preferences, and feedback.
- **M9: Hardening & Polish** — Metrics, load tests, and benchmarks.

---

## Quick Start & Verification

### Prerequisites
- Go 1.27+ installed.

### Run Tests
```bash
go test -v ./...
# Or without cache:
go test -count=1 -v ./...
```

### Build
```bash
go build ./...
```

---

## Project Documentation
All project memory and architectural specifications are maintained in [`docs/`](docs/):
- [`docs/SPECIFICATION.md`](docs/SPECIFICATION.md): Full Project Charter & Engineering Specification (v1.0).
- [`docs/milestones.md`](docs/milestones.md): Active milestones and acceptance gates.
- [`docs/progress.md`](docs/progress.md): Chronological, evidence-backed progress log.
- [`docs/teaching-notes.md`](docs/teaching-notes.md): Pedagogical notes, concepts taught, and rules of engagement.
- [`docs/architecture.md`](docs/architecture.md): High-level system architecture and invariants.
- [`docs/decisions/`](docs/decisions/): Architecture Decision Records (ADRs).

