# NetForge — Milestones & Acceptance Gates

Source of truth: `docs/SPECIFICATION.md` (Section 10).
Milestones are capability gates, not calendar deadlines. Advance only when evidence shows the acceptance gate is met.

---

### M0 — Repository and Learning Contract
- [x] Create repository, README, design notes, configuration strategy.
- [x] Establish a minimal compiling Go application (`go mod init netforge`).
- [x] Run test suite from a clean checkout (`go test -v ./...`).
- **Acceptance Gate:** Met. Clean checkout compiles, tests pass, project contract documented.

---

### M1 — Domain Model and Durable Jobs
- [x] Define Job states and valid state machine transitions.
- [x] Write state transition unit tests (`internal/job/state_test.go`).
- [x] Define `Job` domain entity and `TransitionTo` method (`internal/job/job.go`).
- [x] Implement error categories (transient vs permanent) (`internal/job/errors.go`).
- [x] Define storage contract `Repository` interface (`internal/job/repository.go`).
- [x] Implement in-memory repository structure (`internal/job/memory_repository.go`).
- [ ] Add `sync.RWMutex` thread safety to in-memory repository.
- [ ] Write unit tests for `MemoryRepository` (`internal/job/memory_repository_test.go`).
- [ ] Define `Run` domain model.
- [ ] Implement PostgreSQL access and migrations.
- **Acceptance Gate:** Accepted jobs persist across restart; invalid state transitions are rejected; tests cover the lifecycle.

---

### M2 — Single-Process Worker Pool
- [ ] Implement bounded worker pool (semaphore / channels).
- [ ] Atomic job claiming (`FOR UPDATE SKIP LOCKED`).
- [ ] `context.Context` cancellation and per-task deadlines.
- [ ] Attempt tracking and execution duration.
- [ ] Graceful shutdown (SIGINT/SIGTERM) leaving recoverable state.
- [ ] CLI status inspection commands.
- **Acceptance Gate:** Concurrency is bounded; jobs execute once per successful claim; race tests pass (`go test -race ./...`); shutdown leaves recoverable state.

---

### M3 — Retry and Crash Recovery
- [ ] Bounded exponential backoff with jitter.
- [ ] Time-bounded leases and lease renewal.
- [ ] Abandoned job recovery on lease expiry.
- [ ] Crash-window fault-injection tests.
- **Acceptance Gate:** Transient failures retry within limits; permanent failures terminate; abandoned jobs recover; duplicate attempts do not corrupt state.

---

### M4 — First Useful Research Workflow
- [ ] Source discovery adapter (initial single dependable source path).
- [ ] Safe HTTP retrieval (deadlines, size limits, SSRF controls).
- [ ] Content extraction and text normalization.
- [ ] URL canonicalization and content fingerprint deduplication.
- [ ] Persisted research report with source links.
- **Acceptance Gate:** User submits a research question and receives a persisted report with source provenance; failures are visible.

---

### M5 — Evidence-Backed Analysis
- [ ] Structured claim extraction.
- [ ] Citation association and support verification.
- [ ] Uncertainty and conflict classification (supported / partially supported / conflicting / unsupported).
- [ ] Safe schema validation for model outputs.
- **Acceptance Gate:** Benchmark set produces traceable claims; unsupported claims are flagged; malformed model output is handled safely.

---

### M6 — On-Demand Research Orchestration
- [ ] Bounded query decomposition / planning into non-overlapping subtasks.
- [ ] Fan-out / fan-in execution with dependency graph.
- [ ] Run budgets (max subtasks, sources, duration, token/cost limits).
- [ ] Deterministic fallback plan if LLM planner fails.
- **Acceptance Gate:** Independent subtasks run concurrently; synthesis waits for required results; failed subtasks are represented honestly.

---

### M7 — Daily Technical Briefing
- [ ] Configurable recurring schedule.
- [ ] Topic preferences and time windows.
- [ ] Novelty ranking against prior processed fingerprints.
- [ ] Concise briefing generation and delivery.
- **Acceptance Gate:** Scheduled run executes repeatedly without duplicate briefings or uncontrolled repetition.

---

### M8 — Personal Research Memory & UX
- [ ] Topic history and retained preferences.
- [ ] Report search, listing, and inspection.
- [ ] User feedback controls to correct relevance.
- **Acceptance Gate:** Prior reports inform relevance/deduplication; user can inspect and correct preferences.

---

### M9 — Hardening & Operational Polish
- [ ] Structured logging (`run_id`, `job_id`, `attempt`).
- [ ] Key metrics (queue wait, worker saturation, failure rates).
- [ ] Reproducible load tests and failure injection.
- [ ] Documented recovery behavior, operational limits, demo.
- **Acceptance Gate:** Documented recovery behavior, reproducible setup, measured throughput, known limits, credible demo.

