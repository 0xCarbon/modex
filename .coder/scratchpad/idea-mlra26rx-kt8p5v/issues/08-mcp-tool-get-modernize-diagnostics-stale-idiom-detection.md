# MODEX-008 - MCP Tool: get_modernize_diagnostics (Stale Idiom Detection)

Status: backlog
Priority: P1
Tags: MCP Tool, Diagnostics, Modernization, Go
Depends-On: MODEX-001, MODEX-004
Estimated-Effort: 4 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001, MODEX-004 |
| blocks | MODEX-009 |
| tags | `MCP Tool`, `Diagnostics`, `Modernization`, `Go` |
| estimated_effort | 4 days |

## Goal
Implement the `get_modernize_diagnostics` MCP tool to identify opportunities for modernizing Go code to current idioms and features (Go 1.25+), using `go fix -diff` as the core mechanism.

## Problem
LLMs frequently generate Go code with stale idioms, resisting modern patterns even when directed. This tool will proactively detect such patterns and suggest modern alternatives, guiding LLMs towards generating high-quality, idiomatic Go 1.25+ code.

## Scope
- Define the `get_modernize_diagnostics` MCP tool interface.
- **Use `go fix -diff ./...` as the core mechanism**: This outputs unified diffs showing what would change WITHOUT modifying files. This is the key insight from the go fix research -- preview mode gives us structured diagnostics for free.
- Parse the unified diff output into structured diagnostics:
  ```go
  type ModernizeDiagnostic struct {
      File       string `json:"file"`
      Line       int    `json:"line"`
      Fixer      string `json:"fixer"`       // e.g., "waitgroup", "rangeint", "stringscut"
      MinGo      string `json:"min_go"`      // e.g., "1.25", "1.22", "1.18"
      OldCode    string `json:"old_code"`    // The current code
      NewCode    string `json:"new_code"`    // The suggested replacement
      Explanation string `json:"explanation"` // Why this modernization matters
  }
  ```
- **Handle disabled-by-default modernizers**: Three modernizers (`appendclipped`, `bloop`, `slicesdelete`) are disabled by default due to subtle behavioral differences. Implement two modes:
  - **Safe mode** (default): Only enabled-by-default modernizers.
  - **Aggressive mode**: All modernizers, with warnings about behavioral differences (e.g., `slicesdelete` changes memory zeroing behavior -- security-sensitive code should not use it).
- **Detect `//go:fix inline` annotations** in project dependencies. Scan imported packages for `//go:fix inline` annotations on deprecated functions and surface these as available API migrations.
- **Include version gates per suggestion**: Each diagnostic must report which Go version is required, so the LLM knows if the suggestion is applicable to the current project.
- **Run iteratively until convergence**: The go fix blog explicitly states that applying one fix can create opportunities for another. Run `go fix -diff` in a loop (max 3 passes) until no more changes are produced, and report the cumulative diff.

## All 24 Modernizers (Reference Table)

| Fixer | Min Go | Default | What It Does |
|---|---|---|---|
| `any` | 1.18 | On | `interface{}` -> `any` |
| `appendclipped` | 1.21 | **Off** | Append chains -> `slices.Concat`/`Clone` |
| `bloop` | 1.24 | **Off** | `b.N` loop -> `b.Loop()` |
| `errorsastype` | 1.26 | On | `errors.As(err, &x)` -> `errors.AsType[T](err)` |
| `fmtappendf` | 1.10 | On | `[]byte(fmt.Sprintf(...))` -> `fmt.Appendf` |
| `forvar` | 1.22 | On | Remove redundant `x := x` in range loops |
| `mapsloop` | 1.23 | On | Map loops -> `maps.Copy`/`Clone`/`Collect` |
| `minmax` | 1.21 | On | `if/else` clamping -> `min()`/`max()` |
| `newexpr` | 1.26 | On | Helper `newInt(x)` -> `new(x)` |
| `omitzero` | 1.24 | On | `omitempty` -> `omitzero` for struct fields |
| `plusbuild` | 1.18 | On | Remove obsolete `//+build` tags |
| `rangeint` | 1.22 | On | 3-clause `for` -> `for i := range n` |
| `reflecttypefor` | 1.22 | On | `reflect.TypeOf(T(0))` -> `reflect.TypeFor[T]()` |
| `slicescontains` | 1.21 | On | Loop search -> `slices.Contains` |
| `slicesdelete` | 1.21 | **Off** | `append(s[:i], s[j:]...)` -> `slices.Delete` |
| `slicessort` | 1.21 | On | `sort.Slice` -> `slices.Sort` |
| `stditerators` | 1.22 | On | `Len()/At()` -> range over `.All()` |
| `stringsbuilder` | 1.10 | On | `s += x` in loops -> `strings.Builder` |
| `stringscut` | 1.18 | On | `strings.Index`+slicing -> `strings.Cut` |
| `stringscutprefix` | 1.20 | On | `HasPrefix`+`TrimPrefix` -> `CutPrefix` |
| `stringsseq` | 1.24 | On | `strings.Split` in range -> `SplitSeq` |
| `testingcontext` | 1.24 | On | `context.WithCancel(ctx)` in tests -> `t.Context()` |
| `unsafefuncs` | 1.17 | On | Pointer arithmetic -> `unsafe.Add`/`Slice` |
| `waitgroup` | 1.25 | On | `wg.Add(1)/go func()/defer wg.Done()` -> `wg.Go()` |

## Deliverables
- The `get_modernize_diagnostics` MCP tool is registered.
- It runs `go fix -diff` and parses the unified diff into structured diagnostics.
- Each diagnostic includes the fixer name, version gate, old/new code, and explanation.
- Safe and aggressive modes are supported.
- `//go:fix inline` annotations in dependencies are detected and reported.

## Acceptance Criteria
- When run on code with `ioutil.ReadFile`, it reports the modernization with fixer name and min Go version.
- When run on code with `wg.Add(1); go func() { defer wg.Done(); ... }()`, it suggests `wg.Go()` (Go 1.25+).
- Safe mode skips `appendclipped`, `bloop`, `slicesdelete` with no output for them.
- Aggressive mode includes them with behavioral-change warnings.
- Version gates are respected: a project at `go 1.21` does not get `rangeint` suggestions (requires 1.22).
- Iterative convergence: synergistic fixes are caught in subsequent passes.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Create Go code snippets for each of the 24 modernizers and verify detection.
- Test version gating: set `go 1.21` in `go.mod` and verify that 1.22+ modernizers are not suggested.
- Test safe vs. aggressive mode output.
- Test iterative convergence with a file that has synergistic fix opportunities.
- Test `//go:fix inline` detection with a mock dependency.
- Test against modern, idiomatic Go code to ensure no false positives.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`. Use `testdata/` directories with sample Go modules at different versions.

## Verification
```bash
go test ./internal/diagnostics/modernize/... -v
```

## Non-Goals
- Automatic application of modernization changes (covered by `apply_modernize` MODEX-009).
- Custom analyzers beyond what `go fix` provides (v1 relies on the built-in 24 modernizers).

## Risks and Gaps
- Parsing unified diffs is well-understood but adds a dependency on diff format stability.
- `go fix -diff` is a Go 1.26 feature -- must verify behavior on Go 1.25 (may need fallback).
- `//go:fix inline` scanning requires loading dependency source, which may be slow.

## External References
- [Using go fix to Modernize Go Code (Go Blog)](https://go.dev/blog/gofix) - Authoritative source for `go fix` architecture, iterative convergence, and modernizer design.
- [modernize package](https://pkg.go.dev/golang.org/x/tools/go/analysis/passes/modernize) - All 24 analyzers with documentation.
- [moderngo: Custom ruleguard rules for Go 1.20-1.25+](https://github.com/tphakala/moderngo) - Additional modernization patterns.
- [example-gomodernize: Go code examples for modernization](https://github.com/blck-snwmn/example-gomodernize) - Test fixtures.

## Notes
- `go fix -diff` shares the `go/analysis` framework with `go vet` and `gopls`. An analyzer's `SuggestedFix` contains `TextEdit` entries that drive both the diff output and the actual file modifications.
- The Go team explicitly stated that a primary motivation for modernizers is improving LLM-generated code quality.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
