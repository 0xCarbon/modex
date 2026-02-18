# MODEX-005 - MCP Tool: get_build_diagnostics (Compilation Errors, Vet Warnings, and Race Detection)

Status: backlog
Priority: P1
Tags: MCP Tool, Diagnostics, Build, Go
Depends-On: MODEX-001, MODEX-004
Estimated-Effort: 3 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001, MODEX-004 |
| blocks | _none_ |
| tags | `MCP Tool`, `Diagnostics`, `Build`, `Go` |
| estimated_effort | 3 days |

## Goal
Implement the `get_build_diagnostics` MCP tool to identify and report compilation errors, `go vet` warnings, and **race condition detection** in Go code.

## Problem
LLMs frequently generate Go code with basic compilation blockers such as missing imports, incorrect assignments, and misuse of test packages. The HN thread specifically identified **concurrency bugs (race conditions, deadlocks, goroutine leaks) as the #1 most dangerous LLM failure mode** in Go. This tool will provide precise, actionable feedback covering both build errors and concurrency safety.

## Scope
- Define the `get_build_diagnostics` MCP tool interface, accepting a project path and optional flags.
- **Use `go build -json ./...`** for structured compilation error output (not gopls -- simpler, more reliable, no external dependency).
- **Use `go vet -json ./...`** for structured vet warnings. The `-json` flag produces machine-parseable output with file, line, column, message, and suggested fixes.
- **Integrate `go test -race ./...`** to detect data races. The race detector is one of Go's most powerful tools and directly addresses the #1 LLM concurrency failure mode.
  - Run with a configurable timeout (default 60s) to prevent hangs.
  - Parse race detector output to extract goroutine stacks and shared memory locations.
- Parse and normalize all outputs into a unified diagnostic format:
  ```go
  type BuildDiagnostic struct {
      File     string `json:"file"`
      Line     int    `json:"line"`
      Column   int    `json:"column"`
      Message  string `json:"message"`
      Severity string `json:"severity"` // "error", "warning", "race"
      Category string `json:"category"` // "build", "vet", "race"
      Code     string `json:"code,omitempty"` // e.g., "unusedvariable"
  }
  ```
- Return the structured diagnostic results to the MCP client.

## Deliverables
- The `get_build_diagnostics` MCP tool is registered.
- Compilation errors are reported with structured output from `go build -json`.
- Vet warnings are reported with structured output from `go vet -json`.
- Race conditions are detected via `go test -race` and reported with goroutine stacks.

## Acceptance Criteria
- When run on a Go project with missing imports, it reports the corresponding compilation errors with file/line/column.
- When run on code with `go vet` warnings (e.g., `printf` format mismatches, unreachable code), it reports these warnings.
- When run on code with a data race (e.g., concurrent map access without mutex), it detects and reports the race with both goroutine stacks.
- Diagnostic reports use the unified format above.
- The race detector runs with a timeout and does not hang the tool.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Create a suite of small Go projects with specific compilation errors (undeclared vars, bad imports, type mismatches) and verify structured output.
- Create Go files with `go vet` warnings (printf format bugs, unreachable code, struct tag errors) and verify detection.
- Create a Go test file with a deliberate data race (concurrent map write) and verify the race detector catches it.
- Test timeout behavior: a test that hangs should be killed after the configured timeout.
- Test against a clean Go project to ensure no false positives.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`. Use `testdata/` directories for sample Go projects.

## Verification
```bash
go test ./internal/diagnostics/build/... -v
```

## Non-Goals
- Automatic fixing of build issues (covered by `apply_modernize` for some cases).
- Deep semantic analysis beyond what `go vet` provides (staticcheck integration can come later).
- Full test suite execution (the race detector run is lightweight, not a full test run).

## Risks and Gaps
- `go test -race` requires CGo and may not work in all environments.
- Race detector only finds races that are actually triggered during the test run -- it's not exhaustive.
- The `-json` flag behavior for `go build` may differ across Go versions.

## External References
- [go-tools (Staticcheck): The advanced Go linter](https://github.com/dominikh/go-tools) - Deep insight into diagnostic check implementation.
- [Data Race Detector documentation](https://go.dev/doc/articles/race_detector) - Official guide to Go's race detector.

## Notes
- Focus on common LLM failure modes: `WaitGroup` misuse (`wg.Add` inside goroutine), `test` package imports in non-test files, concurrent map access, channel deadlocks.
- The race detector output format is well-documented and stable -- parse the `WARNING: DATA RACE` blocks with goroutine stack traces.
- Consider adding a `run_race_detector` boolean parameter (default true) since it adds latency.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
