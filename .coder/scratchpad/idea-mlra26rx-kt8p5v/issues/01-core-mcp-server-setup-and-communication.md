# MODEX-001 - Core MCP Server Setup and Communication

Status: backlog
Priority: P0
Tags: MCP, Infrastructure, Go
Depends-On: _none_
Estimated-Effort: 2 days

## Issue Graph

| Key | Value |
|-----|-------|
| depends_on | _none_ |
| blocks | MODEX-002, MODEX-003, MODEX-004, MODEX-005, MODEX-006, MODEX-007, MODEX-008, MODEX-009 |
| tags | `MCP`, `Infrastructure`, `Go` |
| estimated_effort | 2 days |

## Goal
Establish the foundational Model Context Protocol (MCP) server structure in Go, enabling standard input/output (stdio) and Streamable HTTP communication.

## Problem
The 'modex' project requires a robust, Go-native MCP server as its core platform to host all subsequent tools and interact with Large Language Models (LLMs). This foundational setup must correctly integrate the official Go SDK and support specified communication protocols.

## Scope
- Initialize a new Go module (`go mod init github.com/user/modex`).
- Integrate the `github.com/modelcontextprotocol/go-sdk` for MCP server implementation.
- **Use typed/generic tool handlers** (`mcp.AddTool[Input, Output]()`) with automatic JSON schema inference from Go struct `jsonschema` tags. This is the SDK's recommended pattern for type-safe tool definitions.
- Implement server logic to support communication over **stdio** (primary transport for CLI integration).
- Implement server logic to support communication over **Streamable HTTP** (the SDK's recommended HTTP transport -- NOT legacy SSE).
- Configure **rate limiting** (default 30 req/sec), **concurrency limits** (default 64 concurrent requests), and **max registered projects** (default 32) from the start, following cratedex's production-hardened defaults.
- Implement graceful shutdown with signal handling (`SIGINT`, `SIGTERM`).

## Implementation Details

### Typed Tool Handler Pattern (from official SDK)
```go
type SearchInput struct {
    Query string `json:"query" jsonschema:"the search query"`
    Limit int    `json:"limit,omitempty" jsonschema:"max results to return"`
}

mcp.AddTool(server, &mcp.Tool{
    Name:        "search_docs",
    Description: "Search indexed Go documentation",
}, func(ctx context.Context, req *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
    // Implementation
})
```

### Transport Setup
```go
// Stdio
server.ServeTransport(ctx, mcp.StdioTransport{})

// Streamable HTTP (recommended over SSE)
handler := mcp.NewStreamableHTTPHandler(
    func(*http.Request) *mcp.Server { return server }, nil)
http.ListenAndServe(":3838", handler)
```

## Deliverables
- A Go module with the official `go-sdk` integrated.
- The server can be run and accept connections via stdio.
- The server can be run and accept connections via Streamable HTTP.
- Rate limiting and concurrency limits are enforced on HTTP transport.

## Acceptance Criteria
- The `go-sdk` is correctly imported and tool handlers use the typed/generic pattern.
- The server can be run and accept connections via stdio.
- The server can be run and accept connections via Streamable HTTP.
- Rate limiting rejects excess requests with appropriate errors.
- Graceful shutdown completes in-flight requests before exiting.

## Testing Strategy
### Existing Tests
- No existing tests identified.
### New Tests to Write
- Write unit tests for the server initialization and shutdown procedures, verifying graceful startup and termination.
- Develop integration tests using `mcp.NewInMemoryTransports()` (official SDK's in-memory transport for testing) to send MCP messages and assert correct responses.
- Verify rate limiting behavior under concurrent load.
- Verify error handling for malformed requests or connection issues.
### Test Patterns
Standard Go testing conventions using `*_test.go` files and `go test`.

## Verification
```bash
go build ./... && go test ./...
```

## Non-Goals
- Implementation of specific MCP tools (e.g., diagnostics, search_docs).
- Authentication/authorization (can be added later via SDK middleware).

## Risks and Gaps
- Potential compatibility issues with `go-sdk` if not used idiomatically.
- Streamable HTTP transport is newer than SSE; verify client compatibility.

## External References
- [go-sdk: The official Go SDK for Model Context Protocol servers and clients](https://github.com/modelcontextprotocol/go-sdk) - Core constraint; provides typed tool handlers, transport abstractions, middleware.
- [mcp-go: A Go implementation of the Model Context Protocol (MCP)](https://github.com/mark3labs/mcp-go) - Architectural reference (but DO NOT use this SDK; use the official one).
- [mcp-filesystem-server: Go MCP server for filesystem operations](https://github.com/mark3labs/mcp-filesystem-server) - Example of a production Go MCP server.

## Notes
- The official SDK provides `AddSendingMiddleware()` and `AddReceivingMiddleware()` for auth, logging, metrics -- use these instead of hand-rolling middleware.
- `mcp.ServerOptions` supports `Instructions` (string), `Logger` (slog), `PageSize`, `KeepAlive` -- configure all from the start.
- Cratedex binds to loopback (127.0.0.1) by default, requiring `allow_remote=true` for network access. Follow this pattern.

## Scratchpad
- Date:
- Decision notes:
- Blockers:
- Next action:
