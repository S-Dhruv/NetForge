# ADR-0001: Language and Toolchain Selection (Go)

- **Date:** 2026-10-20
- **Status:** Accepted
- **Context:** NetForge requires a language that provides direct access to systems programming concepts (concurrency, networking, database transactions, process lifecycle) within a ~90-day learning scope, while keeping iteration and testing fast.
- **Decision:** Use Go (`go 1.27+`) with the standard library (`testing`, `sync`, `net/http`, `database/sql`) as the primary language.
- **Alternatives Considered:**
  - Python: Rapid iteration, but GIL complicates true multithreading, and dynamic typing can obscure domain contracts.
  - Java: Strong concurrency, but heavy boilerplate, verbose setup, and no installed JDK on current machine.
  - C++: Deep systems control, but manual memory debugging, complex build configurations, and tooling overhead would distract from distributed systems learning.
- **Reasons and Trade-offs:** Go provides first-class concurrency (goroutines, channels, mutexes), a built-in race detector (`go test -race`), fast compilation, a single binary target, and a minimal syntax surface. The trade-off is learning Go syntax and explicit error handling (`if err != nil`), which aligns well with systems resilience goals.
- **Consequences:** All core packages, CLI tools, and tests will be written in Go.

