# NetForge — Teaching Notes & Pedagogical Log

Durable notes on concepts taught, learner feedback, communication protocols, and pedagogical observations.

**Tutor quick start:** Read [`llm-tutor-brief.md`](llm-tutor-brief.md) at the beginning of each new session, then verify the repo and update from [`progress.md`](progress.md).

---

### Working Contract & Learner Preferences
1. **Teaching Protocol:**
   - **Stage 0 (Default):** High-level gist, core mission, constraints, and problem statement only. **NO code snippets and NO prescribed field/schema lists.** Do not spoon-feed struct designs, field names, or architecture layouts—give at most the overarching purpose so the learner reasons through and owns the design decisions themselves.
   - **Stage 1 (Review & Progressive Hints):** Review the learner's design/code, point out edge cases or bugs, and offer graduated hints only if requested or after a genuine attempt. Full solutions only on explicit request.
2. **Design Thinking Log (`docs/design-thinking-log.md`):**
   - **Every time the learner makes a good design decision, update this file** with a new entry: what the idea was, what was strong about it, and what gap remains.
   - **Every time the learner's design can be improved**, add a "What to sharpen" note in the relevant section.
   - Keep entries honest and specific — not just praise, but concrete technical feedback.
   - Update the Summary Table at the bottom of the file whenever a new entry is added.
2. **Product Context:**
   - NetForge is a personal research companion powered by a durable, observable job execution engine.
   - Core invariants: at-least-once execution with idempotency, atomic lease claiming, crash recovery, structured error classification.
3. **Learner Background:**
   - Strong DSA background, familiar with Python and Java, light C++.
   - New to Go. Motivated by systems engineering, concurrency, networking, persistence, and reliability.

---

### Concepts Covered So Far
- **Go Syntax & Layout:**
  - Modules vs packages: `go.mod` defines root; directory names define package names.
  - Exported vs package-private: Initial capital letter (`CanTransition` vs `canTransition`).
  - Strict compiler invariants: No unused imports or variables; `else` must share the closing brace line `} else {`.
  - Type definitions: `type State string`.
- **Testing in Go:**
  - `_test.go` naming convention and `func TestXxx(t *testing.T)`.
  - Table-driven tests using anonymous struct slices (`[]struct { name, from, to, expected }`).
  - Subtests via `t.Run(name, ...)`.
  - Reporting errors via `t.Errorf(...)` without an `assert` keyword.
- **Zsh Autocorrect Quirk:**
  - Zsh flags `./...` as a potential typo for `./..`. Solution: quote `'./...'` or invoke `./internal/job`.
- **Architecture & Object Design:**
  - Global state vs Dependency Injection: global variables pollute unit tests and prevent polymorphic swapping of storage implementations.
  - Interfaces vs Structs: interfaces define behavior contracts (`Repository`); structs hold state (`MemoryRepository`).
  - Errors vs Error Categories: functions return `error` to describe *what happened*; callers evaluate `ErrorCategory` to decide *how to respond*.
  - `context.Context`: unified cancellation propagation across memory and network boundaries.
  - Go types: primitive values (`int64`, `string`) have zero values (`0`, `""`) and cannot be compared to `nil`.
  - Blank identifier `_` for unused keys in `range` loops.

