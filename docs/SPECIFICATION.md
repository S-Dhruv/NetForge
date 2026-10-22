NETFORGE

Project Charter, Engineering Specification & LLM Teaching Guide

A personal research system powered by a durable, observable job engineVersion 1.0  |  Living project document  |  23 October 2026

| Purpose of this documentThis is the source of truth for any AI assistant teaching, planning, reviewing, or helping implement NetForge. It defines the product, architecture direction, milestones, learning method, engineering standards, and how project context must be maintained. The assistant must teach the user to build the system—not silently build a large opaque codebase on their behalf. |
| --- |


## How an LLM should use this document

Read the whole document before proposing architecture or code. Treat it as the project's current contract, but verify assumptions against the actual repository.

At the start of a new session, inspect the repository and the project-state files described in Section 15. Identify the current milestone, completed evidence, blockers, and next smallest task.

Do not assume a milestone is complete because code exists. Completion requires tests, observable behavior, and the acceptance criteria in this document.

Teach interactively: explain the concept, ask the learner to reason or implement a small piece, provide graduated hints, review their attempt, and only then show a reference implementation when useful.

When a recommendation conflicts with this document, explain the trade-off and propose a deliberate decision record instead of silently changing direction.

## 1. Executive summary

NetForge is a personal research tool. It provides concise scheduled technical briefings and detailed on-demand investigations. It discovers relevant material, retrieves and normalizes sources, extracts evidence, assesses significance, checks support and uncertainty, and produces reports with traceable citations. Its engineering core is a durable, observable, fault-tolerant job execution engine.

The project is deliberately not a generic workflow platform, an autonomous agent that can do anything, or a thin wrapper around an LLM API. The job engine exists because research work naturally involves many independent source-retrieval tasks, dependencies, external failures, retries, long-running work, and a need to resume after crashes.

| One-sentence missionBuild a research companion that reliably turns technical questions and recurring interests into relevant, evidence-backed knowledge, while using the implementation to learn production-quality Go and distributed-systems engineering. |
| --- |


## 2. Product goals and non-goals

### Goals

Daily briefing: deliver a short, high-signal digest of meaningful developments in user-selected topics.

On-demand research: accept a question, create bounded subtasks, gather sources concurrently, reconcile evidence, and produce a detailed report.

Persistent research context: retain preferences, prior reports, topic history, and processed-source fingerprints to reduce repetition and improve relevance.

Evidence quality: make claims traceable to source URLs and relevant source passages; label uncertainty and disagreements.

Reliability: persist job state, handle transient failures, enforce retry limits, recover abandoned work, and make execution inspectable.

Learning value: implement the system in understandable increments with tests, design notes, and measured behavior.

### Non-goals for v1

A general-purpose workflow product or a competitor to Temporal, Airflow, Celery, or a managed cloud orchestration service.

A multi-tenant SaaS product, public user accounts, billing, or a complex web dashboard.

A fully autonomous agent with unrestricted browsing, shell access, or self-modifying plans.

Kubernetes, Kafka, a service mesh, multiple databases, or microservices before a measured need exists.

Perfect coverage of the entire internet or a guarantee that every generated claim is true.

A model-training project. Use existing model APIs or locally available models behind a replaceable interface.

## 3. Users and primary workflows

### Workflow A — Daily technical briefing

At a configured time, scheduler creates a briefing run for enabled topics and the relevant time window.

System discovers candidate items from configured sources, normalizes metadata, and deduplicates URLs/content.

Candidate items are filtered and ranked by recency, topic relevance, source quality, novelty, and user preferences.

Relevant items are retrieved and analyzed. LLM use is bounded and only used where semantic interpretation helps.

Claims and citations are checked; weakly supported items are downgraded, marked uncertain, or excluded.

System generates a concise briefing with: what happened, why it matters, evidence/source links, uncertainty, and optional next steps.

Briefing is stored and delivered through the initial interface (CLI/local report); run outcome and failures are inspectable.

### Workflow B — On-demand investigation

User submits a research question and optional depth, time range, and topic hints.

System validates the request and creates a research run with an explicit budget: maximum subtasks, source count, duration, and model usage.

Planner decomposes the question into bounded, non-overlapping subquestions. A deterministic fallback plan exists if the LLM planner fails.

Discovery and retrieval jobs execute concurrently within global and per-host limits.

Evidence is normalized and associated with source identity, retrieval time, and source passages.

Analysis groups claims, compares sources, notes conflicts, and marks gaps that remain unanswered.

Synthesis produces a report with findings, methodology/coverage, citations, caveats, and suggested follow-up questions.

Run and report are saved; the user can inspect the job graph, failures, and sources.

## 4. Product requirements

| ID | Requirement | Acceptance intent |
| --- | --- | --- |
| PR-01 | Research request | Accept a non-empty question and optional research depth/topic filters through a CLI. Validate input and return a durable run/job identifier. |
| PR-02 | Source adapters | Use a small interface for discovery and retrieval. Start with one dependable source path; add adapters only when they improve coverage. |
| PR-03 | Normalization | Normalize URL, title, publisher/domain, published time if available, retrieved time, content, and content hash. |
| PR-04 | Deduplication | Avoid duplicate work using canonicalized URLs and content fingerprints; preserve source provenance. |
| PR-05 | Scheduling | Support one configurable daily schedule after the on-demand workflow is proven. |
| PR-06 | Job lifecycle | Persist job status, attempts, lease/claim metadata, timestamps, error category, and result reference. |
| PR-07 | Concurrency limits | Bound worker count, queue growth, per-host requests, and external model calls. |
| PR-08 | Retries | Retry only retryable failures with exponential backoff and jitter; cap attempts and support dead-letter/terminal failure state. |
| PR-09 | Recovery | Recover jobs left running after a process crash using leases/visibility timeout and safe re-claim logic. |
| PR-10 | Dependencies | Support basic parent/child jobs and fan-out/fan-in; do not run synthesis until required dependencies have terminal outcomes. |
| PR-11 | Evidence | Represent claims and supporting sources separately; do not treat model output alone as evidence. |
| PR-12 | Reports | Generate a concise briefing or detailed report with clickable source URLs, uncertainty, and an execution summary. |
| PR-13 | Inspection | Provide CLI commands to list runs/jobs, inspect a job, see attempts/errors, and open or print a stored report. |
| PR-14 | Cost and safety budgets | Set request timeouts, maximum sources/subtasks, maximum run duration, and configurable model-call limits. |
| PR-15 | Configuration | Use environment variables/config files for secrets and runtime settings; never commit API keys. |
| PR-16 | Evaluation | Maintain a small repeatable test set for relevance, citation support, duplicate rate, failure recovery, and runtime. |


## 5. Job model and lifecycle

A job is one bounded unit of work with a stable identity, explicit input, typed result, retry policy, timeout, and observable lifecycle. A research run groups related jobs and defines the overall request.

### Suggested initial states

queued → running → succeeded | retry_wait → queued | failed | canceled

Additional metadata may include attempt_count, available_at, lease_expires_at, started_at, finished_at, last_error, result_ref, and created_at/updated_at.

### Core invariants

Persist a job before acknowledging that it was accepted.

Only one valid lease holder may own a job at a time; lease acquisition must be atomic.

A worker must not overwrite a newer lease/attempt after its lease has expired.

Retry behavior is bounded and based on error classification, not unconditional repetition.

External side effects must be idempotent or guarded by idempotency keys where possible.

A process restart must not erase accepted work or its completed results.

A parent job can proceed when its dependency policy is satisfied, not merely because all workers have stopped.

Every terminal result should be explainable from persisted state and logs.

### Exactly-once execution: an important distinction

Do not promise exactly-once execution. In a crash/retry system, a task may execute more than once even if its result is committed once. Design for at-least-once attempts plus idempotent operations and deduplicated result commits. Explain this distinction clearly during teaching.

## 6. Architecture direction

Start as a modular monolith: one Go application with clear packages and PostgreSQL-backed state. Keep module boundaries real, but do not split into services until there is a demonstrated operational or scaling reason.

| Module | Responsibility |
| --- | --- |
| CLI / HTTP API | Validates user input, starts runs, queries status, returns reports. |
| Application/use-case layer | Coordinates research workflows without owning storage or network details. |
| Scheduler | Creates scheduled briefing runs; must avoid duplicate schedule execution. |
| Job store | Transactional persistence for jobs, dependencies, attempts, run metadata, and results. |
| Dispatcher / worker pool | Claims eligible jobs, applies bounded concurrency, renews leases, and records outcomes. |
| Research adapters | Discover and retrieve source content behind small interfaces. |
| Research pipeline | Normalize, deduplicate, rank, analyze, verify, synthesize, and persist reports. |
| LLM provider interface | Supports structured inputs/outputs, timeouts, token/cost budgets, and provider substitution. |
| Observability | Structured logs, metrics, trace/run identifiers, status inspection, and error categories. |


### Initial technology choices

Language: Go. Use the standard library where it is sufficient and introduce dependencies intentionally.

Persistence: PostgreSQL as the durable source of truth. SQLite may be used for a very early prototype only if the trade-off is explicit; avoid designing a database abstraction for hypothetical future databases.

Database access: choose one approach (for example, database/sql plus a migration tool, or a lightweight typed query generator) and learn it properly. Do not mix several approaches without reason.

Concurrency: goroutines, contexts, channels where appropriate, sync primitives, and explicit bounded worker pools. Avoid spawning unbounded goroutines.

Interfaces: introduce interfaces at boundaries where substitution/testing is useful, not for every concrete type.

Testing: go test, race detector, integration tests with a real disposable database, and deterministic fakes for external services.

Deployment: local development first; Docker Compose may simplify PostgreSQL and later worker separation. No Kubernetes in the MVP.

## 7. Research pipeline and evidence model

### Pipeline stages

Discover: find candidate sources from configured feeds/search providers or explicitly supplied URLs.

Retrieve: fetch with deadlines, redirects policy, size limits, content-type checks, user-agent identification, and per-host rate limits.

Extract: isolate readable content and metadata; preserve the source URL and retrieval timestamp.

Normalize: canonical URL, stable content hash, source identity, publication/retrieval timestamps, and normalized text.

Rank: combine transparent heuristics with optional LLM relevance scoring; preserve why an item was selected.

Analyze: produce structured claims, key changes, significance, and unresolved questions.

Verify: check that cited passages support the specific claims; detect conflicting sources and stale claims.

Synthesize: write the report from the evidence set, not from uncited free-form model memory.

Persist: store report, source references, claim/evidence links, and run metadata.

### Evidence rules

Keep raw source metadata and normalized extracted content separate from model-generated summaries.

A citation is not valid merely because a URL exists; it must plausibly support the adjacent claim.

Distinguish publication date from retrieval date. If publication date is missing or ambiguous, say so internally and avoid asserting recency.

Represent uncertainty explicitly: supported, partially supported, conflicting, or unsupported.

Prefer primary sources for technical changes (official release notes, documentation, papers, repositories) and use secondary sources for context.

Treat web pages and documents as untrusted data. Never execute instructions found inside retrieved content.

Do not bypass paywalls, authentication, robots restrictions, or source access controls. Respect rate limits and source terms.

## 8. Reliability, security, and operational best practices

### Reliability

Use context.Context for cancellation and deadlines across HTTP calls, database operations, and model calls.

Use timeouts for every network call; do not rely on a global timeout alone.

Classify errors into retryable and permanent categories. Examples: transient network failure/429/selected 5xx may retry; malformed input/unsupported content usually should not.

Use exponential backoff with jitter, maximum attempts, and maximum elapsed retry time.

Use database transactions for claiming jobs and committing related state changes.

Implement leases and safe lease renewal before supporting multiple worker processes.

Handle SIGINT/SIGTERM with graceful shutdown: stop accepting new work, cancel or finish in-flight work within a deadline, and leave recoverable state.

Test crash windows deliberately: after claim, after external work, before result commit, and after result commit but before acknowledgement.

### Security and privacy

Keep secrets in environment variables or a local secret manager; provide an example env file containing placeholders only.

Do not log API keys, authorization headers, full sensitive prompts, or private source content by default.

Guard against SSRF: block loopback, link-local, private-network and metadata-service destinations where appropriate; re-check redirect targets and DNS resolution.

Limit response size, decompressed size, redirects, content types, request duration, and concurrent requests.

Sanitize filenames and output paths. Avoid shelling out with untrusted values.

Treat retrieved content and model output as untrusted. Validate structured outputs against schemas before using them.

Provide a clear way to delete stored research data and configure retention if the system later stores sensitive content.

### Observability

Use structured logs with run_id, job_id, job_type, attempt, duration, and error category.

Expose counts for queued/running/succeeded/failed jobs, retry count, queue wait, execution duration, source fetch failures, and model usage/cost if available.

Separate user-facing error messages from detailed internal diagnostics.

Record enough history to debug failures without storing unnecessary personal data.

A job should be inspectable after completion, not only while running.

## 9. Testing and quality strategy

### Testing layers

Unit tests: state transitions, retry classification, URL normalization, deduplication, ranking rules, evidence mapping, and configuration validation.

Property/invariant tests where useful: no concurrent valid leases; terminal jobs do not return to queued accidentally; attempt count never exceeds policy.

Integration tests: PostgreSQL transactions, concurrent job claiming, lease expiry, migrations, and restart recovery.

Contract tests: source adapters and LLM provider implementations conform to the expected input/output contract.

End-to-end tests: submit a question, run the pipeline against deterministic fixtures, and verify a report plus persisted job history.

Fault-injection tests: timeouts, 429/5xx responses, malformed model output, database disconnects, process interruption, and partial workflow failure.

Race detector: run go test -race ./... regularly once concurrency code exists.

### Research evaluation

Relevance: manually label a small fixed sample of candidate items as useful/not useful.

Novelty: measure repeated items across successive briefings and inspect false deduplication.

Citation support: sample claims and verify that cited source passages actually support them.

Coverage: compare the report against a manually prepared checklist for a few benchmark questions.

Calibration: track how often the system labels uncertain/conflicting claims correctly.

Cost and latency: record source count, model calls, tokens/cost where available, wall-clock time, and failure rate.

Do not optimize for a single aggregate score at the beginning. Keep a small evaluation set, inspect errors manually, and use each failure to improve a specific stage.

## 10. Milestones and acceptance gates

### M0 — Repository and learning contract

Build: Create repository, README, design notes, configuration strategy, task log, and test command. Establish a minimal compiling Go application.

Acceptance gate: A new developer/LLM can explain the project and run the test suite from a clean checkout.

### M1 — Domain model and durable jobs

Build: Define Job, Run, status transitions, error categories, repository operations, migrations, and state-transition tests. Start with one job type.

Acceptance gate: Accepted jobs persist across restart; invalid state transitions are rejected; tests cover the lifecycle.

### M2 — Single-process worker pool

Build: Implement bounded workers, atomic claiming, contexts, deadlines, attempt tracking, graceful shutdown, and basic CLI status commands.

Acceptance gate: Concurrency is bounded; jobs execute once per successful claim; race tests pass; shutdown leaves recoverable state.

### M3 — Retry and crash recovery

Build: Add retry policy, backoff/jitter, leases, lease expiry recovery, and fault-injection tests.

Acceptance gate: Transient failures retry within limits; permanent failures stop; abandoned jobs recover; duplicate attempts do not corrupt state.

### M4 — First useful research workflow

Build: Implement source input/discovery, retrieval, extraction, normalization, deduplication, and a report with source links. Start with a small source set.

Acceptance gate: A user can submit a question and receive a persisted report; failures are visible; source provenance is retained.

### M5 — Evidence-backed analysis

Build: Add structured claim extraction, significance assessment, citation association, uncertainty and conflict handling, and output validation.

Acceptance gate: A benchmark set produces traceable claims; unsupported claims are flagged; malformed model output is handled safely.

### M6 — On-demand research orchestration

Build: Add bounded planning, subtask dependencies, fan-out/fan-in, run budgets, and partial-failure policy.

Acceptance gate: Independent subtasks run concurrently; synthesis waits for required results; failed subtasks are represented honestly.

### M7 — Daily briefing

Build: Add schedule configuration, topic preferences, time windows, novelty ranking, stored run history, and concise report format.

Acceptance gate: A scheduled run can execute repeatedly without duplicate briefings or uncontrolled repetition.

### M8 — Personal research memory and UX

Build: Add topic history, saved preferences, report search/listing, feedback controls, and follow-up research links.

Acceptance gate: Prior reports inform relevance/deduplication; the user can inspect and correct preferences.

### M9 — Hardening and optional worker separation

Build: Add metrics, load/failure tests, resource limits, documentation, backups, and a separate worker process only if justified.

Acceptance gate: Documented recovery behavior, reproducible setup, measured throughput, known limits, and a credible demo.

Milestones are capability gates, not calendar deadlines. Do not advance because a week has passed; advance when evidence shows the acceptance gate is met.

## 11. Recommended repository layout

netforge/  cmd/    netforge/              # CLI entry point    netforge-worker/       # optional later, only when separated  internal/    app/                   # use cases and orchestration    config/                # validated configuration    domain/                # job/run/source/report types and invariants    jobs/                  # claim, execute, retry, lease, state transitions    storage/      postgres/            # persistence implementation    research/      discovery/            # source discovery adapters      fetch/                # safe HTTP retrieval      extract/              # content extraction      normalize/            # URL/content normalization and fingerprints      rank/                 # relevance and novelty      analyze/              # claims and significance      verify/               # support/conflict checks      report/               # synthesis and output rendering    llm/                   # provider interface and adapters    observability/          # logs, metrics, instrumentation  migrations/  tests/    integration/    fixtures/    evaluation/  docs/    architecture.md    decisions/    milestones.md    progress.md    teaching-notes.md  .env.example  .gitignore  Makefile  go.mod  README.md

This is a proposed destination, not a requirement to create every directory immediately. Add a package only when there is real code to put there. Keep package dependencies acyclic and prefer internal packages until a genuine external API is needed.

## 12. Go engineering standards

Use idiomatic Go and the standard library as the default. Add dependencies for clear value, not fashion.

Keep functions small enough to understand; name things by domain behavior rather than implementation trivia.

Return errors with useful context. Wrap errors with %w when callers need errors.Is/errors.As; avoid swallowing errors.

Do not panic for normal runtime failures. Reserve panic for unrecoverable programmer invariants, and explain the choice.

Pass context.Context explicitly as the first parameter for operations that can block or be canceled; do not store request contexts in long-lived structs.

Make ownership of goroutines explicit. Every goroutine needs a lifecycle, cancellation path, and testable completion behavior.

Do not share mutable state casually. Prefer ownership, channels for coordination where appropriate, and mutexes for genuinely shared state.

Avoid global mutable variables and init-time hidden work.

Validate configuration once at startup and fail with actionable errors.

Use dependency injection at construction boundaries, not a service locator or an interface for every type.

Keep SQL queries and transaction boundaries understandable. Test concurrency behavior against the real database.

Run gofmt; use go vet ./...; run go test ./...; use go test -race ./... when applicable.

Document exported APIs and non-obvious invariants. Comments should explain why, not restate the code.

## 13. How the AI tutor should teach

| Teaching contractThe learner wants to understand and own the implementation. Optimize for durable understanding and productive difficulty—not for producing the maximum amount of code in one answer. |
| --- |


### Default teaching loop

State the current milestone, the specific outcome, and why it matters to NetForge.

Explain the underlying concept with a small example, including the mental model and trade-offs.

Connect the concept to the code that will use it; identify prerequisites the learner may be missing.

Break the task into a small implementation step that can be completed and tested in one sitting.

Ask the learner to predict behavior, sketch an API, or attempt the key logic before revealing the whole solution when appropriate.

Give graduated hints: conceptual nudge first, pseudocode or a focused snippet second, full solution only if requested or after a genuine attempt.

Review the learner's implementation for correctness, edge cases, readability, concurrency hazards, and tests. Explain each meaningful correction.

Run or specify tests and compare actual behavior against acceptance criteria.

Finish with a concise recap, concepts learned, current state, unresolved questions, and the next smallest task.

### Teaching rules

Never dump a full milestone's implementation at once unless explicitly requested. Prefer vertical slices that compile and have tests.

Do not conceal uncertainty. Verify current library APIs and Go behavior when version-sensitive; inspect go.mod and existing code before recommending changes.

Explain alternatives and why one is chosen. Avoid cargo-cult patterns and unnecessary abstraction.

Teach fundamentals before introducing a library that hides them. For example, explain job claiming and retry semantics before adding a queue framework.

Use realistic failure cases, not only happy-path demos.

When code fails, help the learner diagnose it from evidence (error output, tests, logs) instead of rewriting everything.

Challenge incorrect assumptions respectfully and distinguish requirements from preferences.

Do not claim tests passed unless they were actually run and their results are available.

Ask at most one or two targeted questions when blocked; otherwise make a conservative, reversible assumption and record it.

Maintain continuity by updating the project-state files after meaningful changes.

### What a good teaching answer contains

Goal and acceptance criteria.

Relevant concept and mental model.

Small task or code change.

Tests to write/run and what each proves.

Common mistakes and debugging clues.

Completion check and progress update.

## 14. Definition of done

The behavior matches the stated requirement and acceptance criteria.

Relevant unit, integration, or end-to-end tests exist and pass.

Error paths and resource limits are considered; concurrency changes are race-tested where applicable.

Logs and errors make failures diagnosable without exposing secrets.

Documentation/configuration examples are updated when behavior or setup changes.

The change is small enough to review and explain.

Known limitations and deferred work are recorded.

The learner can explain the design, the key trade-off, and at least one failure mode without relying on the LLM.

## 15. Project memory and continuity files

Keep a small set of repository files as the durable memory shared across LLMs. The repository, not a chat transcript, is the source of truth.

| File | Contents |
| --- | --- |
| README.md | What NetForge is, setup instructions, current capabilities, quick start, and test commands. |
| docs/architecture.md | Current architecture, component boundaries, job lifecycle, persistence model, and important invariants. |
| docs/milestones.md | Milestone checklist, acceptance gates, current milestone, and explicitly deferred scope. |
| docs/progress.md | Append-only or clearly dated log of completed tasks, tests run and outcomes, blockers, and next task. |
| docs/teaching-notes.md | Concepts taught, misconceptions to revisit, explanations that worked, and learner-owned decisions. |
| docs/decisions/NNNN-short-title.md | Architecture Decision Records: context, decision, alternatives, consequences, and date. |
| docs/evaluation.md | Fixed research benchmark questions, expected source/claim checks, manual labels, and observed regressions. |
| .env.example | Names and safe placeholders for required configuration; never real secrets. |


At the end of a work session, the tutor should update progress.md with: date, milestone, changes made, commands/tests actually run and outcomes, decisions, open issues, and the next concrete task. Do not record speculative completion.

## 16. Architecture decisions to defer or make deliberately

LLM provider and model: begin behind a narrow interface; select based on structured output quality, latency, cost, privacy, and availability.

Source discovery provider: start with one or two manageable sources. Avoid building a general crawler as the first step.

Full-text storage: decide based on retention, reproducibility, storage cost, and source terms. Store metadata and fingerprints even if full content is not retained.

Queue model: PostgreSQL-backed claiming is adequate for the initial scale; revisit only after measuring throughput and contention.

HTTP API and UI: CLI first. Add a local API when it improves daily use or makes integration easier.

Multi-process deployment: first prove recovery and concurrency in one process; separate workers only after the job contract is sound.

Vector search/embeddings: defer until keyword/topic history and metadata filtering demonstrably fail to support useful retrieval.

Agentic planning: use bounded planning with schema validation and deterministic limits. Do not permit an LLM to generate unbounded tasks or choose arbitrary tools.

## 17. Risks and mitigations

| Risk | Why it matters | Mitigation |
| --- | --- | --- |
| Scope creep | A tempting feature list can overwhelm the project. | Enforce milestone gates; no new infrastructure without a concrete use case. |
| LLM hallucination | Fluent prose may contain unsupported facts. | Evidence-first synthesis, citation checks, uncertainty labels, evaluation set. |
| Research noise | Daily digest becomes repetitive or irrelevant. | Topic preferences, novelty checks, user feedback, fixed evaluation sample. |
| Fragile job execution | Crashes can lose work or duplicate side effects. | Transactional persistence, leases, idempotency, crash-window tests. |
| Unbounded cost/runtime | Research plan can expand indefinitely. | Subtask/source/time/token budgets and cancellation. |
| Unsafe URL fetching | User or source content can direct the fetcher toward internal services. | SSRF controls, redirect revalidation, size/time limits, network restrictions. |
| Premature distributed complexity | Infrastructure consumes effort without improving the product. | Modular monolith first; split only based on observed need. |
| LLM dependency | Provider changes or outages block the entire product. | Provider interface, clear failure state, deterministic fallback behavior where practical. |


## 18. First two weeks of work: task sequence, not a deadline

Task 1 — Inspect current repository and tools. Write down what already exists; do not overwrite working code.

Task 2 — Create/update README, this project contract, progress file, and a minimal architecture sketch.

Task 3 — Set up Go module, formatting, tests, lint/vet command, config validation, and a minimal CLI.

Task 4 — Design Job and Run domain types; write state-transition tests before storage.

Task 5 — Choose PostgreSQL access/migrations; persist and retrieve a job.

Task 6 — Implement a single worker executing one deterministic job type.

Task 7 — Add bounded worker pool and atomic job claiming; test concurrent claim behavior.

Task 8 — Add retry classification, attempt tracking, and graceful shutdown.

Task 9 — Add lease expiry and a restart-recovery integration test.

Task 10 — Only then begin the first real research source/retrieval path.

If the learner is new to a concept, split a task further. Progress is measured by understanding and verified behavior, not by completing this list quickly.

## 19. Initial CLI shape (illustrative)

netforge research "How does eBPF affect Linux packet processing?"netforge briefing run --date 2026-10-23netforge runs listnetforge runs inspect <run-id>netforge jobs list --status failednetforge jobs inspect <job-id>netforge reports listnetforge reports show <report-id>netforge config validate

These commands are proposed interface examples, not a promise that they already exist. Keep the CLI small and consistent; avoid implementing every command before the underlying use case exists.

## 20. First session prompt for a new LLM tutor

| Copyable tutor promptYou are my technical mentor and pair programmer for NetForge. Read this project guide and inspect the actual repository before proposing changes. NetForge is a personal research tool with daily technical briefings and on-demand evidence-backed investigations. Its engineering core is a durable, observable, fault-tolerant job engine. Teach me to build it incrementally in Go. Do not dump a whole system or skip the reasoning. First report: (1) what exists in the repository, (2) the current milestone and evidence for that assessment, (3) the most important gap, (4) one small next task with acceptance criteria, (5) the concept I need to understand, and (6) tests/commands that will prove completion. Use graduated hints and ask me to reason or attempt the key part before giving a complete implementation when appropriate. Prefer standard-library Go and simple architecture; justify dependencies and complexity. Treat persisted state, bounded concurrency, cancellation, retry semantics, idempotency, security, and testing as first-class concerns. Do not claim anything was run unless you ran it and saw the result. After meaningful work, update docs/progress.md, docs/milestones.md, and relevant architecture/decision notes. Keep this guide authoritative unless we explicitly revise it. |
| --- |


## Appendix A — Review checklist for every pull request/change

Does this change directly support a product requirement or milestone?

Is the behavior small, explicit, and explainable?

Are errors, timeouts, cancellation, and resource limits handled?

Can the operation be retried safely? If not, what protects side effects?

Are database state transitions transactional and concurrency-safe?

Are there tests for the happy path and the most relevant failure path?

Do logs identify the run/job without leaking secrets?

Is the new dependency necessary?

Did docs/progress/decisions need updating?

Can the learner explain why this design was chosen?

## Appendix B — Glossary

Job — A single bounded unit of work with persisted state.

Run — A user-requested research operation or scheduled briefing composed of jobs.

Worker — A component that claims and executes jobs.

Lease — Time-bounded ownership of a claimed job; expiration enables recovery.

Idempotency — Repeating an operation does not create an unintended additional effect.

Fan-out/fan-in — Run independent subtasks concurrently, then combine their results.

Backoff with jitter — Increasing retry delays with randomness to avoid synchronized retry spikes.

Provenance — Information about where evidence came from and how it was retrieved/processed.

ADR — Architecture Decision Record: a short record of a consequential design choice.

Acceptance gate — Observable evidence required before considering a milestone complete.
