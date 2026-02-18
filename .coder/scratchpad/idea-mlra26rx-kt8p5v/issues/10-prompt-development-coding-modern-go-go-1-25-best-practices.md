# MODEX-010 - Prompt Development: coding_modern_go (Go 1.25+ Best Practices)

Status: done
Priority: P0
Tags: Prompt Engineering, Go, Best Practices, LLM Guidance
Depends-On: _none_
Estimated-Effort: 3 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | _none_ |
| blocks | _none_ |
| tags | `Prompt Engineering`, `Go`, `Best Practices`, `LLM Guidance` |
| estimated_effort | 3 days |

## Goal
Develop comprehensive content for the `coding_modern_go` prompt to guide LLMs on generating modern, idiomatic Go 1.25+ code, addressing common failure modes. This prompt should also prescribe a workflow for using modex tools effectively.

## Problem
LLMs often generate Go code with stale idioms, concurrency bugs, version skew, dependency hallucination, and general non-idiomatic patterns. A well-crafted, detailed prompt is essential to steer LLMs towards generating high-quality, up-to-date Go code that adheres to best practices and avoids common pitfalls.

## Scope
The prompt should follow cratedex's `coding_modern_rust.md` structure (857 lines) adapted for Go. Required sections:

### 1. Tool Usage Workflow (gopls `instructions.md` pattern)
Prescribe how the LLM should use modex tools in sequence:
- **Always start** with `get_go_version_info` to understand the project's Go version and available features.
- **Use `search_docs`** before importing unfamiliar packages to verify they exist and check their API.
- **Run `get_diagnostics`** after writing code to catch compilation errors, vet warnings, and races.
- **Run `get_modernize_diagnostics`** to check for stale idioms in generated code.

### 2. Complete Modernizer Table
Embed the full table of all 24 `go fix` modernizers with version gates, so the LLM knows which patterns to use from the start rather than relying on post-hoc fixing:

| Pattern | Min Go | Use This Instead |
|---|---|---|
| `interface{}` | 1.18 | `any` |
| `for i := 0; i < n; i++` | 1.22 | `for i := range n` |
| `sort.Slice(s, func(i,j int) bool{...})` | 1.21 | `slices.Sort(s)` |
| `wg.Add(1); go func() { defer wg.Done(); ... }()` | 1.25 | `wg.Go(func() { ... })` |
| `errors.As(err, &target)` | 1.26 | `errors.AsType[*T](err)` |
| ... (all 24) | | |

### 3. Concurrency Safety (Dedicated Section - #1 Priority)
The HN thread identified this as the most dangerous LLM failure mode. Must cover:
- **WaitGroup patterns**: Always use `wg.Go()` (Go 1.25+). Never put `wg.Add(1)` inside a goroutine. The `waitgroup` modernizer catches this.
- **Channel deadlock prevention**: Always have a receiver before sending on unbuffered channels. Use `select` with `default` or `context.Done()` for cancellation.
- **Goroutine leak prevention**: Every goroutine must have a shutdown path. Use `context.Context` for cancellation. Use `t.Context()` in tests (Go 1.24+).
- **Concurrent map access**: Always use `sync.Map` or mutex-protected maps. Never read/write a map from multiple goroutines without synchronization.
- **Race detector**: Recommend running `go test -race` after any concurrent code changes.

### 4. GODEBUG Awareness
When suggesting Go version upgrades in `go.mod`:
- Document that bumping `go 1.21` to `go 1.22` changes loop variable scoping (GODEBUG: `loopvar`).
- Document that each Go version has default GODEBUG values; upgrading changes behavior.
- Recommend using `godebug` blocks in `go.mod` for gradual migration.

### 5. Standard Sections (from cratedex template)
- New language features (Go 1.22: range over int, loop var scoping; Go 1.23: range over func; Go 1.24: `t.Context()`, `omitzero`; Go 1.25: `wg.Go()`, generic aliases; Go 1.26: `errors.AsType`, `new(expr)`)
- Blessed standard library packages (prefer `slices`, `maps`, `cmp` over manual loops)
- Error handling (wrap with `%w`, use `errors.Is`/`errors.As`, define sentinel errors)
- Testing (table-driven tests, `t.Parallel()`, `t.Context()`, `testing/fstest`, `testing/slogtest`)
- Project setup (`go.mod` best practices, directory structure, build tags)
- Anti-patterns with before/after examples
- Dependency guidance (prefer stdlib, verify packages exist via `search_docs` before importing)

### 6. Dependency Hallucination Prevention
- Explicitly tell the LLM to use `search_docs` to verify package existence before importing.
- List commonly hallucinated packages and their real alternatives.
- Emphasize that Go's stdlib is comprehensive -- most tasks don't need external dependencies.

## Deliverables
- A comprehensive `coding_modern_go` prompt file (target: 600-900 lines).
- Coverage of all sections listed above.
- Embedded modernizer table with version gates.
- Dedicated concurrency safety section.
- Tool usage workflow prescription.

## Acceptance Criteria
- The prompt covers all 24 modernizers with version gates.
- The concurrency section includes concrete before/after examples for WaitGroup, channels, context, and sync.Map.
- The tool workflow section prescribes when to call each modex tool.
- GODEBUG implications are documented for version upgrades.
- The prompt is well-structured with clear section headers for an AI agent to navigate.
- The prompt is at least 400 lines (substantial guidance, not superficial).

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Manual review by Go experts to ensure technical accuracy and idiomatic correctness.
- Validate that all 24 modernizer entries match the official `go fix` documentation.
- Internal dogfooding with LLMs (not part of this issue's scope, but the eventual verification method).
### Test Patterns
N/A, this is a content-based task rather than code implementation.

## Verification
```bash
# Verify prompt file exists and has substantial content
wc -l prompts/coding_modern_go.md && test $(wc -l < prompts/coding_modern_go.md) -gt 400
```

## Non-Goals
- Technical implementation of how this prompt is injected or managed by the MCP server.
- Empirical testing of the prompt with LLMs (this is a content creation task).

## Risks and Gaps
- Prompt content must be updated with each Go release (every 6 months).
- Balancing comprehensiveness with token efficiency -- prompt may be too long for some LLM context windows.

## External References
- [go-concurrency-guide: Practical concurrency guide in Go](https://github.com/luk4z7/go-concurrency-guide) - Common concurrency issues and patterns.
- [go-concurrency-patterns: Concurrency patterns in Go](https://github.com/lotusirous/go-concurrency-patterns) - Well-established concurrency patterns to recommend.
- [moderngo: Custom ruleguard rules for Go 1.20-1.25+](https://github.com/tphakala/moderngo) - Concrete examples of modernization rules.
- [cratedex coding_modern_rust.md](~/Programming/AITOOLS/cratedex/prompts/coding_modern_rust.md) - Template/reference (857 lines covering Rust Edition 2024, APIs, crates, anti-patterns, concurrency, testing).
- [gopls MCP instructions.md](https://go.dev/gopls/features/mcp) - Workflow prescription pattern for AI clients.
- [Using go fix to Modernize Go Code (Go Blog)](https://go.dev/blog/gofix) - Complete modernizer reference.

## Notes
- The Go team explicitly stated: LLMs "tended to produce Go code in a style similar to the mass of Go code used during training" and "often refused to use the newer ways even when explicitly told." This prompt is the direct countermeasure.
- Model after cratedex's structure but adapted for Go idioms. Key difference: Go has no ownership/borrowing/lifetimes sections but needs much heavier concurrency coverage.
- Consider making the prompt version-aware: when served via MCP, inject the project's Go version so the LLM only sees applicable patterns.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
