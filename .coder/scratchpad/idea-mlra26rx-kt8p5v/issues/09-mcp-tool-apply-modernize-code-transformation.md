# MODEX-009 - MCP Tool: apply_modernize (Code Transformation)

Status: backlog
Priority: P1
Tags: MCP Tool, Modernization, Refactoring, Go
Depends-On: MODEX-001, MODEX-008
Estimated-Effort: 2 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001, MODEX-008 |
| blocks | _none_ |
| tags | `MCP Tool`, `Modernization`, `Refactoring`, `Go` |
| estimated_effort | 2 days |

## Goal
Implement the `apply_modernize` MCP tool to automatically apply code modernization transformations, with support for selective fixer application and iterative convergence.

## Problem
After identifying opportunities to modernize Go code, LLMs or users need a direct way to apply these changes to the codebase. The `apply_modernize` tool will provide this capability, streamlining the refactoring process and ensuring adherence to modern Go idioms.

## Scope
- Define the `apply_modernize` MCP tool interface:
  ```go
  type ApplyModernizeInput struct {
      ProjectPath string   `json:"project_path" jsonschema:"required,path to the Go project"`
      Fixers      []string `json:"fixers,omitempty" jsonschema:"specific fixers to apply (default: all enabled)"`
      Aggressive  bool     `json:"aggressive,omitempty" jsonschema:"include disabled-by-default fixers"`
      DryRun      bool     `json:"dry_run,omitempty" jsonschema:"preview changes without applying (returns diff)"`
  }
  ```
- **Support selective fixer application**: Accept an optional list of specific fixers to run (e.g., `["waitgroup", "rangeint"]`). This maps to `go fix` flags like `-waitgroup=true -rangeint=true -any=false ...`.
- **Implement iterative convergence**: Run `go fix` in a loop (max 3 passes) until no more changes are produced. The go fix blog explicitly notes that applying one fix can create opportunities for another (synergistic fixes).
- **Support dry-run mode**: When `dry_run=true`, use `go fix -diff` to return the diff without modifying files. This lets the LLM preview and confirm before applying.
- **Report applied changes**: After application, return a summary of what changed (files modified, fixers that triggered, line counts).
- **Verify compilation after application**: Run `go build ./...` after applying fixes to ensure no compilation errors were introduced.

## Deliverables
- The `apply_modernize` MCP tool is registered.
- Selective fixer application works via the `fixers` parameter.
- Iterative convergence applies synergistic fixes automatically.
- Dry-run mode returns diffs without modifying files.
- Post-application compilation check ensures no breakage.

## Acceptance Criteria
- When `apply_modernize` is run on a file with outdated `io/ioutil` usage, the file is updated to use modern `os` and `io` packages.
- When run with `fixers=["waitgroup"]`, only WaitGroup patterns are modernized.
- Iterative convergence: a file where fixing `forvar` enables `rangeint` applies both in one invocation.
- Dry-run mode returns the diff without modifying any files.
- Post-application `go build` passes (no compilation errors introduced).
- No unintended side effects from running `go fix` multiple times (idempotent after convergence).

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Create isolated Go code files with common outdated patterns and verify transformation.
- Test selective fixer application: only the specified fixers should trigger.
- Test iterative convergence: create a file with synergistic fix opportunities and verify both fixes are applied.
- Test dry-run mode: verify files are not modified and diff output is correct.
- Test post-application compilation: verify `go build` passes after fixes.
- Test idempotence: run `apply_modernize` twice and verify no changes on the second run.
- Test error case: a file where `go fix` would introduce a compilation error (should be caught by the post-build check).
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`. Use `testdata/` directories with sample Go files.

## Verification
```bash
go test ./internal/diagnostics/modernize/... -run TestApplyModernize -v
```

## Non-Goals
- Undo/rollback mechanisms (users should use version control).
- Custom analyzers beyond the 24 built-in modernizers.

## Risks and Gaps
- `go fix` modifying files in-place requires careful handling in concurrent scenarios.
- The post-build check adds latency but is essential for safety.
- Selective fixer flags may vary across Go versions.

## External References
- [Using go fix to Modernize Go Code (Go Blog)](https://go.dev/blog/gofix) - Documents iterative convergence and selective fixer flags.
- [gopatch: Refactoring and code transformation tool for Go](https://github.com/uber-go/gopatch) - Insights into programmatic code manipulation.
- [rf: A refactoring tool for Go](https://github.com/rsc/rf) - Core principles of automated code refactoring.

## Notes
- Go 1.26's `go fix` supports per-fixer enable/disable flags. The exact flag format is `-fixername=bool`.
- The convergence loop should log each pass's changes so the LLM can understand the cascade.
- Consider git-diffing the working tree before and after to produce a clean summary of all changes.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
