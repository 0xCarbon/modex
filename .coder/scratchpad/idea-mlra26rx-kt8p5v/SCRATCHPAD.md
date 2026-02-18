# Idea-to-Issue Research Run: idea-mlra26rx-kt8p5v

- repo_root: /home/fcc/Programming/modex
- repo_path: .
- max_issues: 10
- iterations: 3
- web_research: true
- validate_ideas: true
- validation_mode: poc

## Pointers
```text
## Project: modex — A Go-focused MCP Server for LLM-Assisted Go Development

### Goal
Build "cratedex for Go" — an MCP server that makes LLMs great at writing modern, idiomatic Go code. Named "modex" (module expert / modernization expert).

### Background Research Summary

#### What cratedex does (the Rust equivalent at ~/Programming/AITOOLS/cratedex):
- **MCP server** (HTTP + stdio transport) built in Rust with rmcp SDK
- **SQLite + FTS5** full-text search index for Rust documentation
- **Per-project registration** with background indexing (parses rustdoc JSON)
- **Tools**: register_project, search_docs (with crate/kind/parent filters, FTS5 queries), get_diagnostics (compact summary), get_build_diagnostics (cargo clippy/check output), get_outdated_diagnostics (cargo-outdated), get_security_diagnostics (cargo-audit), list_crates
- **Resources**: cratedex://logs
- **Prompts**: `coding_modern_rust` — comprehensive 850-line guide covering Edition 2024, new stable APIs, blessed crate stack, ownership patterns, error handling, async patterns, anti-patterns, project setup with lint config
- **Architecture**: engine/ (server.rs 3296 LOC, docs.rs 2181 LOC, diagnostics.rs 602 LOC), file watcher for live diagnostics, config via TOML + env vars, cross-platform service install

#### Existing Go MCP Landscape (gaps modex should fill):
1. **godoc-mcp** (mrjoshuak): Wraps `go doc` for documentation lookup. Single tool, no diagnostics, no modernization.
2. **gopls MCP** (official, experimental v0.20.0+): 12 tools (diagnostics, context, search, references, rename, vulncheck). Read-only/introspective. Shares LSP session memory. No doc indexing, no modernization guidance.
3. **mcp-gopls** (hloiseau): 14 tools including test execution, coverage, code manipulation via LSP. Richest existing server but no FTS doc search, no modernization prompt.
4. **go-dev-mcp** (fpt): Validation (go vet, build, mod tidy, gofmt). GitHub integration. No doc search.
5. **mcp-language-server** (isaacphi): Generic LSP-MCP bridge (definition, references, diagnostics, hover, rename). Language-agnostic.

**GAP**: No existing Go MCP server combines ALL of: (a) indexed doc search with FTS, (b) comprehensive diagnostics, (c) modernization guidance (go fix/modernize), (d) version-aware idiom recommendations, (e) a "coding modern Go" prompt equivalent. modex fills this gap.

#### Go fix / Modernize (Critical Feature):
Go 1.26 (Feb 2026) completely rewrote `go fix` atop the Go analysis framework. 24 modernizers available:
- Version-gated (respects go.mod `go` directive per file)
- `//go:fix inline` for custom API migrations
- Shares analyzers with go vet and gopls
- Key fixers: any, rangeint, minmax, waitgroup, errorsastype, stringscut, slicescontains, omitzero, testingcontext, etc.
- Can run programmatically via go/analysis framework or shell out to `go fix -diff`

#### LLM Go Failure Modes (from deep-research-report.md):
- **Compilation blockers**: Missing/unused imports, := vs =, test package misuse (+29.34% compile rate from simple repair)
- **Concurrency bugs**: WaitGroup misuse, deadlocks, goroutine leaks
- **Version skew**: Models deny existence of new APIs (e.g., WaitGroup.Go in Go 1.25)
- **Dependency hallucination**: 19.7% fictitious packages in studies
- **Stale idioms**: LLMs produce code matching training data, resist modern patterns

#### HN Thread Insights (item 47049479):
- LLMs "tended to produce Go code in a style similar to the mass of Go code used during training"
- Even when explicitly directed, LLMs resist using newer features
- Concurrent code is "over-simplified and missing error and edge case handling"
- go fix modernizers were created partly to improve LLM training data quality
- Links to Coccinelle (C), ruff/pyupgrade (Python), lebab (JS) as analogues in other ecosystems

#### Technical Stack:
- **Language**: Go 1.25+ (installed: go1.25.6)
- **MCP SDK**: github.com/modelcontextprotocol/go-sdk (official, maintained with Google)
- **Transport**: stdio (primary) + HTTP/SSE
- **Storage**: SQLite with FTS5 (via modernc.org/sqlite or mattn/go-sqlite3)
- **Go tooling integration**: go doc, go vet, go fix, go list, gopls, govulncheck
- **Testing**: standard testing + testify, golangci-lint for linting
- **Architecture**: Similar to cratedex but Go-native

### Proposed MCP Tools:
1. **register_project** — Register a Go module, trigger background indexing
2. **search_docs** — FTS5-powered search across indexed Go documentation (stdlib + deps)
3. **get_diagnostics** — Compact summary (build errors, vet warnings, lint issues)
4. **get_build_diagnostics** — Detailed go build/vet/staticcheck output
5. **get_outdated_diagnostics** — Outdated dependencies (go list -m -u)
6. **get_security_diagnostics** — govulncheck results
7. **get_modernize_diagnostics** — go fix analysis (what can be modernized, version-gated)
8. **apply_modernize** — Apply specific go fix transformations
9. **get_go_version_info** — Module Go version, toolchain, unlocked modernizers
10. **list_modules** — List workspace modules and dependencies

### Proposed MCP Prompts:
1. **coding_modern_go** — Comprehensive guide (like coding_modern_rust.md) covering Go 1.25/1.26 features, modern idioms, blessed packages, common mistakes, concurrency patterns, error handling, testing patterns

### Proposed MCP Resources:
1. **modex://logs** — Server logs
2. **modex://project/{path}/go.mod** — Module definition

### Key Design Principles:
- Version-aware: All recommendations respect the project's go.mod version
- Modernization-first: Proactively surface go fix opportunities
- Defensive: Address known LLM failure modes (import management, concurrency, version skew)
- Lightweight: Fast startup, background indexing, minimal dependencies
- Composable: Works alongside gopls MCP or other Go MCP servers
```

## Clarifications
```text
(none provided)
```

## Step: analyze_chunk_01
- timestamp: 2026-02-18T00:12:04.175Z
- status: running

## Step: analyze_chunk_01
- timestamp: 2026-02-18T00:12:20.279Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/analyze-chunk-01.json

## Step: aggregate_pointer_analysis
- timestamp: 2026-02-18T00:12:20.280Z
- status: running

## Step: aggregate_pointer_analysis
- timestamp: 2026-02-18T00:12:38.818Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/analysis-brief.json

## Context Gather Complete
- timestamp: 2026-02-18T00:12:38.818Z
- chunks_analyzed: 1
- problem_spaces: 9

## Step: collect_web_references
- timestamp: 2026-02-18T00:12:48.906Z
- status: running

## Step: collect_web_references
- timestamp: 2026-02-18T00:18:38.077Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/web-references.json

## Step: plan_validation_tracks
- timestamp: 2026-02-18T00:18:46.334Z
- status: running

## Step: plan_validation_tracks
- timestamp: 2026-02-18T00:19:14.810Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/validation-plan.json

## Step: execute_validation_tracks
- timestamp: 2026-02-18T00:19:14.810Z
- status: running

## Step: draft_issue_backlog_01
- timestamp: 2026-02-18T00:28:57.509Z
- status: running

## Step: draft_issue_backlog_01
- timestamp: 2026-02-18T00:30:14.895Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-01.json

## Iteration 1 Draft
- timestamp: 2026-02-18T00:30:14.895Z
- agent: gemini
- candidate_issues: 10
- draft_json: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-01.json

## Step: review_issue_backlog_01
- timestamp: 2026-02-18T00:30:14.896Z
- status: running

## Step: review_issue_backlog_01
- timestamp: 2026-02-18T00:30:36.073Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/review-01.json

## Iteration 1 Critique
- timestamp: 2026-02-18T00:30:36.073Z
- agent: gemini
- must_fix: 1
- should_fix: 1
- reference_gaps: 1
- validation_gaps: 0
- testing_gaps: 0
- review_json: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/review-01.json

## Step: draft_issue_backlog_02
- timestamp: 2026-02-18T00:30:36.074Z
- status: running

## Step: draft_issue_backlog_02
- timestamp: 2026-02-18T00:31:58.876Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-02.json

## Iteration 2 Draft
- timestamp: 2026-02-18T00:31:58.876Z
- agent: gemini
- candidate_issues: 10
- draft_json: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-02.json

## Step: review_issue_backlog_02
- timestamp: 2026-02-18T00:31:58.876Z
- status: running

## Step: review_issue_backlog_02
- timestamp: 2026-02-18T00:32:19.571Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/review-02.json

## Iteration 2 Critique
- timestamp: 2026-02-18T00:32:19.571Z
- agent: gemini
- must_fix: 0
- should_fix: 1
- reference_gaps: 0
- validation_gaps: 0
- testing_gaps: 0
- review_json: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/review-02.json

## Step: draft_issue_backlog_03
- timestamp: 2026-02-18T00:32:19.571Z
- status: running

## Step: draft_issue_backlog_03
- timestamp: 2026-02-18T00:33:28.261Z
- status: completed
- agent: gemini
- artifact: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-03.json

## Iteration 3 Draft
- timestamp: 2026-02-18T00:33:28.261Z
- agent: gemini
- candidate_issues: 10
- draft_json: .coder/scratchpad/idea-mlra26rx-kt8p5v/steps/draft-03.json

## Step: spec_publish
- timestamp: 2026-02-18T00:34:04.939Z
- status: running

## Step: spec_publish
- timestamp: 2026-02-18T00:34:04.941Z
- status: completed
- issueCount: 10

## Generated Issues
- timestamp: 2026-02-18T00:34:04.941Z
- MODEX-001: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/01-core-mcp-server-setup-and-communication.md
- MODEX-002: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/02-documentation-indexing-with-sqlite-fts5.md
- MODEX-003: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/03-mcp-tool-search-docs-implementation.md
- MODEX-004: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/04-mcp-tool-get-diagnostics-orchestrator-for-all-diagnostics.md
- MODEX-005: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/05-mcp-tool-get-build-diagnostics-compilation-errors-and-vet-warnings.md
- MODEX-006: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/06-mcp-tool-get-outdated-diagnostics-outdated-dependencies.md
- MODEX-007: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/07-mcp-tool-get-security-diagnostics-vulnerability-scanning.md
- MODEX-008: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/08-mcp-tool-get-modernize-diagnostics-stale-idiom-detection.md
- MODEX-009: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/09-mcp-tool-apply-modernize-code-transformation.md
- MODEX-010: .coder/scratchpad/idea-mlra26rx-kt8p5v/issues/10-prompt-development-coding-modern-go-go-1-25-best-practices.md
- manifest: .coder/scratchpad/idea-mlra26rx-kt8p5v/manifest.json
