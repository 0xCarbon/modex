# MODEX-004 - MCP Tool: get_diagnostics (Orchestrator for all Diagnostics)

Status: backlog
Priority: P0
Tags: MCP Tool, Diagnostics, Orchestration
Depends-On: MODEX-001
Estimated-Effort: 2 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001 |
| blocks | MODEX-005, MODEX-006, MODEX-007, MODEX-008 |
| tags | `MCP Tool`, `Diagnostics`, `Orchestration` |
| estimated_effort | 2 days |

## Goal
Develop a central `get_diagnostics` MCP tool that orchestrates and consolidates results from all specific diagnostic tools, plus a first-class `get_go_version_info` tool that reports Go version context and unlocked modernizers.

## Problem
To effectively address various Go code issues (compilation, outdated dependencies, security, modernization, version skew), LLMs need a single, coherent diagnostic report. The `get_diagnostics` orchestrator will fulfill this by calling and aggregating results from individual diagnostic tools **concurrently** with independent error handling.

## Scope
- Define the `get_diagnostics` MCP tool interface, accepting parameters relevant for comprehensive diagnostics (e.g., project path, specific checks to run).
- **Run all sub-diagnostics concurrently** using `errgroup.Group` with independent error handling -- a failing sub-diagnostic should not block others (follow cratedex's pattern of parallel `cargo-audit` + `cargo-outdated`).
- **Implement `get_go_version_info` as a first-class standalone MCP tool** (not just aggregated data). This was identified by the HN thread and go fix research as the single highest-value diagnostic tool. It should:
  - Parse `go.mod` to extract the `go` directive and `toolchain` directive.
  - Report the installed Go toolchain version (`go version`).
  - **List all 24 modernizers with their version gates**, marking which are unlocked for this project's declared Go version.
  - Show what additional modernizers would unlock at each version bump (e.g., "upgrading to go 1.24 unlocks: `bloop`, `omitzero`, `stringsseq`, `testingcontext`").
  - Report GODEBUG implications of upgrading the `go` directive.
- Define a unified, structured output format for aggregated diagnostic results.

## Version Info Output Example

```json
{
  "go_mod_version": "1.22",
  "toolchain_version": "go1.25.6",
  "unlocked_modernizers": [
    {"name": "any", "min_go": "1.18", "description": "interface{} -> any"},
    {"name": "minmax", "min_go": "1.21", "description": "if/else -> min()/max()"},
    {"name": "rangeint", "min_go": "1.22", "description": "3-clause for -> range n"}
  ],
  "upgrade_opportunities": [
    {
      "target_version": "1.23",
      "new_modernizers": ["mapsloop"],
      "godebug_changes": []
    },
    {
      "target_version": "1.24",
      "new_modernizers": ["bloop", "omitzero", "stringsseq", "testingcontext"],
      "godebug_changes": ["httpmuxgo121 removed"]
    },
    {
      "target_version": "1.25",
      "new_modernizers": ["waitgroup"],
      "godebug_changes": []
    }
  ]
}
```

## Deliverables
- The `get_diagnostics` MCP tool is registered and runs sub-diagnostics concurrently.
- The `get_go_version_info` MCP tool is registered as a standalone tool.
- All diagnostic results are merged into a single, structured JSON output.
- Sub-diagnostic failures are reported inline without blocking the overall report.

## Acceptance Criteria
- `get_diagnostics` runs all sub-diagnostics concurrently and returns within a reasonable time.
- If one sub-diagnostic fails (e.g., `govulncheck` not installed), the others still return results with the failure noted.
- `get_go_version_info` correctly identifies the Go version and lists unlocked/locked modernizers.
- `get_go_version_info` correctly shows upgrade opportunities with GODEBUG implications.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Write unit tests for the `get_diagnostics` handler, mocking sub-diagnostics to verify concurrent execution and result aggregation.
- Test partial failure: mock one sub-diagnostic to fail, verify others complete and the failure is reported.
- Test `get_go_version_info` with `go.mod` files at different Go versions (1.18, 1.21, 1.22, 1.24, 1.25, 1.26) and verify the correct modernizers are listed as unlocked.
- Test upgrade opportunity calculation with GODEBUG changes.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`.

## Verification
```bash
go test ./internal/diagnostics/... -v
```

## Non-Goals
- Implementation details of the individual diagnostic tools (covered in MODEX-005, -006, -007, -008).
- Deep analysis of individual diagnostic findings (focus on orchestration).

## Risks and Gaps
- The modernizer version gate table (24 entries) must be kept in sync with Go releases.
- GODEBUG implications are complex and change across Go versions.

## External References
- [golangci-lint: Fast linters runner for Go](https://github.com/golangci/golangci-lint) - Example of concurrent linter orchestration.
- [Go 1.26 modernize package](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/modernize) - Authoritative source for the 24 modernizers and their version gates.
- [GODEBUG documentation](https://tip.golang.org/doc/godebug) - Reference for GODEBUG settings and their lifecycle.

## Notes
- The complete modernizer table with version gates was catalogued by the go fix research agent. Embed it as a Go data structure for the version info tool.
- Consider making `get_go_version_info` the first thing an LLM calls when starting work on a Go project.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
