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
- **Transports** — stdio (for Claude Desktop / claude CLI) and HTTP/SSE
- **Pure-Go SQLite** — no CGo, no system libraries required

## Tools

| Tool | Description |
|------|-------------|
| `register_project` | Register a Go project directory for indexing. Validates `go.mod` exists and starts background indexing of stdlib + all project dependencies. |
| `get_index_status` | Check indexing progress for a registered project. Returns phase, total/indexed/skipped/failed counts. |
| `ping` | Health check — returns `pong`. |

## Prompts

| Prompt | Description |
|--------|-------------|
| `coding_modern_go` | System prompt for writing modern, idiomatic Go 1.25+ code. Covers module layout, error handling, concurrency patterns, generics, and common pitfalls. |

## Installation

**Prerequisites:** Go 1.25+

```bash
go install github.com/0xCarbon/modex/cmd/modex@latest
```

Or build from source:

```bash
git clone https://github.com/0xCarbon/modex
cd modex
go build ./cmd/modex
```

## Usage

### stdio transport (claude CLI / Claude Desktop)

```bash
modex
# or explicitly:
modex --transport stdio
```

### HTTP transport

```bash
modex --transport http --addr 127.0.0.1:3838
```

### Database location

The index is stored at `~/.cache/modex/modex.db` by default. Override with `--db`:

```bash
modex --db /path/to/modex.db
```

## Claude Desktop configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "modex": {
      "command": "modex"
    }
  }
}
```

## Claude CLI configuration

Add to your MCP settings or run directly:

```bash
claude --mcp-server "modex:modex"
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

3. Once `phase` is `ready`, the model can search docs to verify APIs before using them (via MODEX-003 `search_docs` — coming soon).

## Architecture

```
cmd/modex/          main entry point, flag parsing, transport setup
internal/
  db/               SQLite wrapper, FTS5 schema, auto-sync triggers
  docs/             go/doc extraction, background indexer, progress tracking
  server/           MCP server, middleware, tool handlers
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
- [x] MODEX-010: `coding_modern_go` prompt
- [ ] MODEX-003: `search_docs` tool
- [ ] MODEX-004: `get_diagnostics` orchestrator
- [ ] MODEX-005: `get_build_diagnostics`
- [ ] MODEX-006: `get_outdated_diagnostics`
- [ ] MODEX-007: `get_security_diagnostics`
- [ ] MODEX-008: `get_modernize_diagnostics`
- [ ] MODEX-009: `apply_modernize`

## License

MIT
