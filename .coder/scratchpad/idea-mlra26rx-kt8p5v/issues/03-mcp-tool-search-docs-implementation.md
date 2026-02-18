# MODEX-003 - MCP Tool: search_docs Implementation

Status: backlog
Priority: P0
Tags: MCP Tool, Documentation, Search
Depends-On: MODEX-001, MODEX-002
Estimated-Effort: 2 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001, MODEX-002 |
| blocks | _none_ |
| tags | `MCP Tool`, `Documentation`, `Search` |
| estimated_effort | 2 days |

## Goal
Implement the `search_docs` MCP tool to allow LLMs to query the indexed Go documentation using full-text search capabilities, with multiple query modes and filtering options.

## Problem
LLMs need a direct, programmatic interface to access the rich Go documentation indexed in MODEX. The `search_docs` tool will serve as this interface, enabling LLMs to find relevant information quickly and accurately, thereby improving code generation quality and reducing reliance on internal knowledge.

## Scope
- Define the `search_docs` MCP tool using the typed handler pattern from MODEX-001.
- **Implement 4 query modes** (following cratedex's proven design):
  - `auto` (default): Detects `.` separator for symbol paths (e.g., `fmt.Println`) and switches between text and symbol mode automatically.
  - `text`: Tokenize + prefix match for general documentation search.
  - `symbol`: Optimized for Go symbol paths (`package.Symbol`, `Type.Method`).
  - `fts5`: Raw FTS5 query passed directly to SQLite MATCH for power users. FTS5 columns: `package_path`, `item_name` (3x BM25), `parent_name` (2x BM25), `signature` (1.5x BM25), `doc_text`.
- **Implement dependency depth filtering**:
  - `max_depth=0`: Workspace/project packages only.
  - `max_depth=1` (default when `project_path` is set): Project + direct dependencies.
  - `max_depth=null`: All transitive dependencies.
- **Implement parent filtering**: Filter methods/fields by parent type name (e.g., `parent="Reader"` to find all `io.Reader` methods, `parent="Decoder"` for `json.Decoder` methods).
- **Implement kind filtering**: Filter by symbol kind (`func`, `type`, `method`, `interface`, `struct`, `const`, `var`).
- **Implement package filtering**: Restrict results to specific packages (e.g., `packages=["encoding/json", "fmt"]`).
- Support pagination with `limit` (default 10, max configurable) and `offset`.

## Tool Interface

```go
type SearchDocsInput struct {
    Query      string   `json:"query" jsonschema:"required,the search query"`
    ProjectPath string  `json:"project_path,omitempty" jsonschema:"optional project scope"`
    Packages   []string `json:"packages,omitempty" jsonschema:"optional package filters"`
    Kinds      []string `json:"kinds,omitempty" jsonschema:"optional kind filters (func/type/method/interface/struct/const/var)"`
    Parent     string   `json:"parent,omitempty" jsonschema:"filter methods by parent type name"`
    Mode       string   `json:"mode,omitempty" jsonschema:"query mode: auto/text/symbol/fts5 (default: auto)"`
    MaxDepth   *int     `json:"max_depth,omitempty" jsonschema:"max dependency depth (0=project, 1=+direct deps, null=all)"`
    Limit      int      `json:"limit,omitempty" jsonschema:"max results (default 10)"`
    Offset     int      `json:"offset,omitempty" jsonschema:"pagination offset"`
}
```

## Deliverables
- The `search_docs` MCP tool is registered with typed handler and discoverable by MCP clients.
- All 4 query modes work correctly.
- Depth, parent, kind, and package filtering work correctly.
- Results include `has_more` indicator for pagination.

## Acceptance Criteria
- A search for `fmt.Println` in `auto` mode returns the `fmt.Println` documentation.
- A search for `parent="Reader"` returns methods on `io.Reader`.
- A search with `max_depth=0` returns only project-local symbols.
- A raw FTS5 query like `item_name:Marshal OR item_name:Unmarshal` returns both symbols.
- Results are paginated with correct `has_more` indicators.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Write unit tests for query normalization in each mode (auto-detection of `.`, tokenization, prefix matching).
- Test each filtering dimension independently and in combination.
- Test BM25 ranking: `item_name` matches should rank above `doc_text` matches.
- Test pagination: verify `offset`/`limit` work correctly and `has_more` is accurate.
- Test edge cases: empty queries, no results, special characters.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`.

## Verification
```bash
go test ./internal/tools/... -run TestSearchDocs -v
```

## Non-Goals
- Semantic/embedding-based search (FTS5 with BM25 is sufficient for v1).
- Caching search results (FTS5 is fast enough).

## Risks and Gaps
- Query normalization across modes adds complexity.
- BM25 weighting may need tuning based on real-world usage patterns.

## External References
- [engram: Persistent memory system for AI coding agents](https://github.com/Gentleman-Programming/engram) - Working example of an MCP server with FTS5 search.
- [chronicle: Fast Go CLI with SQLite, FTS5 search, and MCP server](https://github.com/harperreed/chronicle) - Another practical MCP + FTS5 implementation.
- Cratedex `search_docs` implementation in `server.rs` -- supports all 4 modes, depth filtering, parent filtering, pagination.

## Notes
- Cratedex's `auto` mode detects `::` for Rust paths; our equivalent detects `.` for Go paths.
- Consider exposing the FTS5 `highlight()` function to return snippets with matched terms highlighted.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
