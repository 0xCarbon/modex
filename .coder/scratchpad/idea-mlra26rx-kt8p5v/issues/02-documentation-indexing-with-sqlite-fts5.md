# MODEX-002 - Documentation Indexing with SQLite FTS5

Status: backlog
Priority: P0
Tags: Documentation, FTS5, Go, Indexing
Depends-On: MODEX-001
Estimated-Effort: 3 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | MODEX-001 |
| blocks | MODEX-003 |
| tags | `Documentation`, `FTS5`, `Go`, `Indexing` |
| estimated_effort | 3 days |

## Goal
Implement a full-text search (FTS5) index using SQLite for comprehensive Go documentation, covering both the standard library and project dependencies.

## Problem
Large Language Models often suffer from 'version skew' and 'dependency hallucination' when generating Go code. Providing a version-aware, indexed documentation source is crucial to mitigate these issues and ensure LLMs produce accurate and up-to-date code.

## Scope
- Set up an SQLite database with WAL mode for crash safety and concurrent read access.
- Enable the FTS5 extension for full-text search.
- **Use `go/doc` and `go/packages` programmatically** for structured doc extraction -- NOT shelling out to `go doc` CLI. The `go/doc` package gives structured access to package documentation, type information, and function signatures, which is far more reliable than parsing CLI text output.
- Crawl Go standard library documentation using `go/packages` to load and `go/doc` to extract.
- Crawl documentation for project dependencies by inspecting `go.mod` and loading packages from the module cache.
- Populate the FTS5 index with collected documentation, ensuring version awareness via the `go` directive in `go.mod`.
- **Implement package deduplication** using `SHA256(module_path || version)` hash to prevent re-indexing identical packages across projects.
- **Implement background indexing** with progress tracking (% complete, current package, ETA) and cancellation support via `context.Context`.

## FTS5 Schema Design (Modeled After Cratedex)

```sql
CREATE TABLE docs (
    id            INTEGER PRIMARY KEY,
    package_path  TEXT NOT NULL,       -- e.g., "encoding/json", "github.com/foo/bar"
    package_hash  TEXT UNIQUE,         -- SHA256(module_path||version) for dedup
    go_version    TEXT,                -- Go version from go.mod
    item_name     TEXT NOT NULL,       -- e.g., "Marshal", "Decoder", "ErrTooLarge"
    kind          TEXT NOT NULL,       -- "func", "type", "method", "const", "var", "interface"
    parent_name   TEXT,                -- "Decoder" for a method on Decoder
    parent_kind   TEXT,                -- "struct" or "interface"
    signature     TEXT,                -- Full type signature for search ranking
    doc_text      TEXT                 -- Doc comment text
);

CREATE VIRTUAL TABLE docs_fts USING fts5(
    package_path,
    item_name,                         -- 3x BM25 weight (most important)
    parent_name,                       -- 2x BM25 weight
    signature,                         -- 1.5x BM25 weight
    doc_text,
    content='docs',
    content_rowid='id',
    tokenize='porter unicode61'        -- Stemming + Unicode support
);
```

## Deliverables
- An SQLite database file is created with WAL mode and FTS5 enabled.
- Standard library documentation is successfully indexed with structured schema.
- Documentation for project dependencies (from `go.mod`) is indexed.
- Background indexing runs with progress tracking and cancellation.
- Duplicate packages are not re-indexed across projects.

## Acceptance Criteria
- The FTS5 schema matches the design above with proper column weighting.
- Standard library documentation is fully indexed.
- Documentation for at least one external Go module dependency is successfully indexed.
- FTS5 queries return relevant results (e.g., `SELECT * FROM docs_fts WHERE docs_fts MATCH 'item_name:WaitGroup'`).
- Re-registering a project with unchanged dependencies skips already-indexed packages.
- Indexing can be cancelled mid-flight without corrupting the database.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Write unit tests for the `go/doc` extraction pipeline: verify that package docs, function signatures, type methods, and constants are correctly parsed into the schema.
- Develop integration tests to ensure that the SQLite database is correctly initialized with FTS5 and that the indexing process populates the table with expected data.
- Create tests to validate BM25-weighted FTS5 queries return results ranked by relevance (item_name matches ranked above doc_text matches).
- Test package deduplication: indexing the same package twice should not create duplicate rows.
- Test cancellation: starting and cancelling indexing should leave the DB in a consistent state.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`.

## Verification
```bash
go test ./internal/docs/... -v
```

## Non-Goals
- Implementation of the `search_docs` MCP tool (covered in MODEX-003).
- Real-time file watching for live re-indexing (can be added later).

## Risks and Gaps
- Performance bottlenecks during documentation crawling and indexing for large projects with many transitive dependencies.
- `go/packages` load times can be slow for large dependency trees -- consider caching loaded package data.
- CGo dependency for `mattn/go-sqlite3` -- consider `modernc.org/sqlite` (pure Go) as alternative.

## External References
- [engram: Persistent memory system for AI coding agents](https://github.com/Gentleman-Programming/engram) - Complete implementation of SQLite + FTS5 + MCP server for AI agents.
- [go-doc-bundler: Go utility that crawls documentation with full-text search](https://github.com/BaseMax/go-doc-bundler) - Documentation crawling and bundling with FTS.
- [go-sqlite3-fts5: Enable FTS5 extension with go-sqlite3](https://github.com/knaka/go-sqlite3-fts5) - Critical technical dependency for enabling FTS5.
- Cratedex `docs.rs` (2181 LOC) at `~/Programming/AITOOLS/cratedex/src/engine/docs.rs` - Reference implementation for FTS5 schema, doc crawling pipeline, deduplication, and progress tracking.

## Notes
- Cratedex uses `cargo +nightly rustdoc --output-format json` for structured docs. The Go equivalent is `go/doc` + `go/packages` which gives us `doc.Package` with structured access to all symbols.
- Consider `go/doc/comment` (Go 1.19+) for rendering doc comments to clean text.
- Database access should use `spawn_blocking` equivalent (goroutine pool or dedicated DB goroutine) to avoid blocking the async server.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
