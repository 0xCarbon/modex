# MODEX-006 - MCP Tool: get_outdated_diagnostics (Outdated Dependencies)

Status: backlog
Priority: P2
Tags: MCP Tool, Diagnostics, Dependencies, Go
Depends-On: MODEX-001, MODEX-004
Estimated-Effort: 2 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001, MODEX-004 |
| blocks | _none_ |
| tags | `MCP Tool`, `Diagnostics`, `Dependencies`, `Go` |
| estimated_effort | 2 days |

## Goal
Implement the `get_outdated_diagnostics` MCP tool to identify and report outdated Go module dependencies within a project.

## Problem
LLMs can suffer from 'dependency hallucination' and often suggest or use outdated dependencies, leading to compatibility issues, security vulnerabilities, or sub-optimal code. This tool will provide version-aware diagnostics to help LLMs maintain an up-to-date dependency graph.

## Scope
- Define the `get_outdated_diagnostics` MCP tool interface.
- Integrate with the `go list -m -u all` command to list all direct and indirect dependencies along with their available updates.
- Parse the output of `go list` to extract information about outdated modules, including current version, latest available version, and potentially semantic versioning details.
- Format the outdated dependency information into a structured diagnostic report, indicating severity and actionable advice.
- Return the structured results to the MCP client.

## Deliverables
- The `get_outdated_diagnostics` MCP tool is registered.
- When run on a Go project with outdated direct dependencies, it reports them.
- Reports include the current version and the latest available version for each outdated module.

## Acceptance Criteria
- The `get_outdated_diagnostics` MCP tool is registered.
- When run on a Go project with outdated direct dependencies, it reports them.
- When run on a Go project with outdated indirect dependencies, it reports them.
- Reports include the current version and the latest available version for each outdated module.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Create a set of sample Go projects with `go.mod` files configured with known outdated dependencies (both direct and indirect).
- Write integration tests to run `get_outdated_diagnostics` on these projects and assert that the tool correctly identifies all outdated modules and their update information.
- Test a project with all up-to-date dependencies to ensure no false positives.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`.

## Research Questions
- What are the best practices for efficiently parsing `go list` output for complex dependency graphs?
- How can we provide context or recommendations for updating dependencies beyond just the latest version (e.g., security patches, major version changes)?

## Verification
```bash
go test ./internal/diagnostics/outdated/... -v
```

## Non-Goals
- Automatic updating of dependencies (user or LLM decision).
- Deep analysis of dependency conflicts (focus on simple outdated status).

## Risks and Gaps
- Potential for `go list` output format to change, breaking parsing logic.
- Identifying 'outdated' can be nuanced (e.g., pre-release versions, specific branch usage).

## External References
- [gomod: Go modules analysis tool](https://github.com/Helcaraxan/gomod) - Dedicated tool for analyzing Go modules.
- [depgraph: Go module dependency graph analysis tool](https://github.com/ldemailly/depgraph) - Helps visualize and analyze the dependency graph.

## Notes
Ensure the tool respects the `go.mod` file for version resolution.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
