# modex

**modex** (module expert / modernization expert) is an MCP server that makes LLMs great at writing modern, idiomatic Go code.

It indexes Go standard library and project dependency documentation into a local SQLite FTS5 database, so LLMs can verify that APIs exist before using them — eliminating dependency hallucination, version skew, and stale idioms.

## Why

LLMs often generate Go code that:
- References functions or types that don't exist in the installed version
- Uses deprecated patterns superseded by newer stdlib additions
- Invents method signatures from memory rather than checking the actual API

modex gives the model a local, searchable index of real documentation — the same docs `go doc` would show — so it can look before it writes.

## Features

- **Full-text search** over stdlib + project dependency docs (FTS5, porter stemming)
- **Background indexing** with progress tracking — no blocking the model
- **Dedup** — packages with unchanged content/version are skipped on re-index
- **Diagnostics orchestration** — build/outdated/security/modernize checks via one tool
- **Modernization apply mode** — run `go fix` changes with dry-run support
- **Transports** — stdio and streamable HTTP (recommended: run as persistent HTTP service)
- **Pure-Go SQLite** — no CGo, no system libraries required

## Tools

| Tool | Description |
|------|-------------|
| `ping` | Health check — returns `pong`. |
| `register_project` | Register a Go project directory for indexing. Validates `go.mod` exists and starts background indexing of stdlib + all project dependencies. |
| `get_index_status` | Check indexing progress for a registered project. Returns phase, total/indexed/skipped/failed counts. |
| `reindex_project` | Restart indexing for a registered project (cancels any active indexing first). |
| `search_docs` | Search indexed docs with modes `auto`, `text`, `symbol`, or `fts5`, plus optional package/kind/parent filters. |
| `get_diagnostics` | Run diagnostics on a project. Categories: `build`, `outdated`, `security`, `modernize` (all categories by default). |
| `apply_modernize` | Apply `go fix` modernizations with optional selective fixers, `aggressive`, and `dry_run` modes. |

## Prompt Asset

`prompts/coding_modern_go.md` contains a system prompt for writing modern, idiomatic Go code with modex.

## Installation

**Prerequisites:** Go 1.26+ (modex uses `go fix` modernizers added in 1.26; it can index projects built with any Go version)

Build from source:

```bash
git clone https://github.com/0xCarbon/modex
cd modex
go build -o modex ./cmd/modex
```

Or install directly:

```bash
go install github.com/0xCarbon/modex/cmd/modex@latest
```

## Usage

modex is designed to run as a persistent HTTP service — start it once and connect from any number of Claude sessions.

### Start the server

```bash
modex --transport http --addr 127.0.0.1:3838
```

The MCP endpoint is served at the root path, e.g. `http://127.0.0.1:3838/mcp`.

### stdio transport

For single-session use (e.g. Claude Desktop on macOS/Windows):

```bash
modex --transport stdio
```

### Database location

The index is stored at `~/.cache/modex/modex.db` by default. Override with `--db`:

```bash
modex --db /path/to/modex.db
```

## Claude Code configuration

Add modex as a global MCP server so it's available in every project:

```bash
claude mcp add --transport http modex http://127.0.0.1:3838/mcp
```

Or edit `~/.claude.json` directly — add to the top-level `mcpServers`:

```json
{
  "mcpServers": {
    "modex": {
      "type": "http",
      "url": "http://127.0.0.1:3838/mcp"
    }
  }
}
```

## Claude Desktop configuration

For stdio transport, add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "modex": {
      "command": "modex"
    }
  }
}
```

## Workflow

1. Register your project once:
   ```
   register_project(project_path="/path/to/your/project")
   ```

2. Indexing runs in the background. Check progress:
   ```
   get_index_status(project_path="/path/to/your/project")
   ```

3. Once `phase` is `ready`, search docs before writing code:
   ```
   search_docs(query="bytes.Buffer.Write", mode="symbol")
   ```

4. Run diagnostics when needed:
   ```
   get_diagnostics(project_path="/path/to/your/project", categories=["build","security"])
   ```

5. Apply modernizations (or preview with `dry_run=true`):
   ```
   apply_modernize(project_path="/path/to/your/project", dry_run=true)
   ```

## Architecture

```
cmd/modex/          main entry point, flag parsing, transport setup
internal/
  db/               SQLite wrapper, FTS5 schema, auto-sync triggers
  diagnostics/      build, outdated, security, modernize orchestration
  docs/             go/doc extraction, background indexer, progress tracking
  server/           MCP server, tool handlers
prompts/            coding_modern_go.md system prompt
```

**Storage:** SQLite with FTS5 virtual table backed by a `docs` content table. Auto-sync triggers keep FTS5 in sync with INSERT/DELETE/UPDATE operations without any manual management.

**Indexing pipeline:**
1. Enumerate stdlib and project packages (lightweight `NeedName|NeedModule`)
2. For each non-deduped package: full load with syntax, extract via `go/doc`, insert in transaction

**Dedup:** SHA256 hash of `packagePath@moduleVersion`. Local packages (no version) use a content fingerprint of their source files instead.

## Development

```bash
# Build
go build ./...

# Fast tests (skips integration tests that index stdlib ~30s)
go test -short ./...

# All tests
go test ./... -timeout 10m

# Specific packages
go test ./internal/db/... -v
go test ./internal/docs/... -v
go test ./internal/server/... -v
```

## Roadmap

- [x] MODEX-001: Core MCP server (ping, rate limiting, concurrency middleware)
- [x] MODEX-002: Documentation indexing (SQLite FTS5, go/doc extraction, background indexer)
- [x] MODEX-003: `search_docs` tool
- [x] MODEX-004: `get_diagnostics` orchestrator
- [x] MODEX-005: build diagnostics (via `get_diagnostics` category `build`)
- [x] MODEX-006: outdated diagnostics (via `get_diagnostics` category `outdated`)
- [x] MODEX-007: security diagnostics (via `get_diagnostics` category `security`)
- [x] MODEX-008: modernize diagnostics (via `get_diagnostics` category `modernize`)
- [x] MODEX-009: `apply_modernize`
- [x] MODEX-010: `coding_modern_go` prompt

## License

MIT
