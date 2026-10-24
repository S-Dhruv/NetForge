# NetForge — Teaching Notes & Pedagogical Log

Durable notes on concepts taught, learner feedback, communication protocols, and pedagogical observations.

---

### Working Contract & Learner Preferences
1. **Teaching Protocol:**
   - **Stage 0 (Default):** Explanations, architectural reasoning, requirements, edge cases, mental models, and conceptual questions. **NO code snippets.**
   - **Stage 1 (On Request / Struggle):** Progressive hints, pseudocode, or small targeted snippets only when the learner explicitly requests or after a genuine attempt.
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

