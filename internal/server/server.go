package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"

	"github.com/0xCarbon/modex/internal/db"
	"github.com/0xCarbon/modex/internal/diagnostics"
	"github.com/0xCarbon/modex/internal/docs"
)

const (
	defaultRateLimitPerSecond   = 3 * 10
	defaultRateLimitBurst       = defaultRateLimitPerSecond
	defaultMaxConcurrentRequest = 1 << 6
	defaultMaxProjects          = 32
)

// App holds server-wide state including the database and registered projects.
type App struct {
	DB       *db.DB
	mu       sync.RWMutex
	projects map[string]*ProjectState
}

// ProjectState tracks a registered project and its background indexer.
type ProjectState struct {
	Path    string
	Indexer *docs.Indexer
	Cancel  context.CancelFunc
}

func rateLimitMiddleware(limiter *rate.Limiter) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if !limiter.Allow() {
				return nil, errors.New("rate limit exceeded")
			}
			return next(ctx, method, req)
		}
	}
}

func concurrencyMiddleware(limit int) mcp.Middleware {
	sem := make(chan struct{}, limit)
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				return next(ctx, method, req)
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
}

type pingArgs struct{}

func pingHandler(_ context.Context, _ *mcp.CallToolRequest, _ pingArgs) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "pong"},
		},
	}, nil, nil
}

type registerProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

func (app *App) registerProjectHandler(_ context.Context, _ *mcp.CallToolRequest, args registerProjectArgs) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(args.ProjectPath) == "" {
		return toolError("project_path is required"), nil, nil
	}

	absPath, err := filepath.Abs(args.ProjectPath)
	if err != nil {
		return toolError("invalid path: %v", err), nil, nil
	}

	// Validate go.mod exists.
	if _, err := os.Stat(filepath.Join(absPath, "go.mod")); err != nil {
		return toolError("no go.mod found at %s: %v", absPath, err), nil, nil
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	// Check if already registered.
	if ps, ok := app.projects[absPath]; ok {
		snap := ps.Indexer.Progress()
		return toolJSON("project already registered", snap), nil, nil
	}

	if len(app.projects) >= defaultMaxProjects {
		return toolError("max registered projects (%d) reached", defaultMaxProjects), nil, nil
	}

	// Create and launch indexer.
	idx := docs.NewIndexer(app.DB, absPath)
	ctx, cancel := context.WithCancel(context.Background())
	app.projects[absPath] = &ProjectState{
		Path:    absPath,
		Indexer: idx,
		Cancel:  cancel,
	}

	go func() {
		if err := idx.Run(ctx); err != nil {
			slog.Error("indexer failed", "project", absPath, "err", err)
		}
	}()

	snap := idx.Progress()
	return toolJSON("project registered, indexing started", snap), nil, nil
}

type getIndexStatusArgs struct {
	ProjectPath string `json:"project_path"`
}

func (app *App) getIndexStatusHandler(_ context.Context, _ *mcp.CallToolRequest, args getIndexStatusArgs) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.ProjectPath)
	if err != nil {
		return toolError("invalid path: %v", err), nil, nil
	}

	app.mu.RLock()
	ps, ok := app.projects[absPath]
	app.mu.RUnlock()

	if !ok {
		return toolError("project not registered: %s", absPath), nil, nil
	}

	snap := ps.Indexer.Progress()
	return toolJSON("index status", snap), nil, nil
}

type reindexProjectArgs struct {
	ProjectPath string `json:"project_path"`
}

func (app *App) reindexProjectHandler(_ context.Context, _ *mcp.CallToolRequest, args reindexProjectArgs) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.ProjectPath)
	if err != nil {
		return toolError("invalid path: %v", err), nil, nil
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	ps, ok := app.projects[absPath]
	if !ok {
		return toolError("project not registered: %s", absPath), nil, nil
	}

	// Cancel existing indexer and replace with a fresh one.
	ps.Cancel()
	idx := docs.NewIndexer(app.DB, absPath)
	ctx, cancel := context.WithCancel(context.Background())
	app.projects[absPath] = &ProjectState{
		Path:    absPath,
		Indexer: idx,
		Cancel:  cancel,
	}

	go func() {
		if err := idx.Run(ctx); err != nil {
			slog.Error("reindexer failed", "project", absPath, "err", err)
		}
	}()

	snap := idx.Progress()
	return toolJSON("project reindexing started", snap), nil, nil
}

type getDiagnosticsArgs struct {
	ProjectPath string                  `json:"project_path"`
	Categories  []diagnostics.Category  `json:"categories,omitempty"`
}

func (app *App) getDiagnosticsHandler(ctx context.Context, _ *mcp.CallToolRequest, args getDiagnosticsArgs) (*mcp.CallToolResult, any, error) {
	if strings.TrimSpace(args.ProjectPath) == "" {
		return toolError("project_path is required"), nil, nil
	}

	absPath, err := filepath.Abs(args.ProjectPath)
	if err != nil {
		return toolError("invalid path: %v", err), nil, nil
	}

	orch := &diagnostics.Orchestrator{ProjectPath: absPath}
	diags, err := orch.Run(ctx, args.Categories)
	if err != nil {
		return toolError("diagnostics failed: %v", err), nil, nil
	}
	return toolJSON("diagnostics", diags), nil, nil
}

// New returns a configured MCP server with rate limiting, concurrency limiting,
// and documentation indexing tools.
func New(database *db.DB) *mcp.Server {
	app := &App{
		DB:       database,
		projects: make(map[string]*ProjectState),
	}

	s := mcp.NewServer(
		&mcp.Implementation{Name: "modex", Version: "0.2.0"},
		&mcp.ServerOptions{Logger: slog.Default()},
	)
	s.AddReceivingMiddleware(rateLimitMiddleware(rate.NewLimiter(
		rate.Every(time.Second/defaultRateLimitPerSecond),
		defaultRateLimitBurst,
	)))
	s.AddReceivingMiddleware(concurrencyMiddleware(defaultMaxConcurrentRequest))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "ping",
		Description: "Health check tool; returns pong.",
	}, pingHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "register_project",
		Description: "Register a Go project for documentation indexing. Validates go.mod exists and starts background indexing of stdlib + project dependencies.",
	}, app.registerProjectHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_index_status",
		Description: "Get indexing progress for a registered Go project. Returns phase, total/indexed/skipped/failed counts.",
	}, app.getIndexStatusHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "reindex_project",
		Description: "Restart indexing for an already-registered Go project. Cancels any in-progress indexing and starts a fresh run.",
	}, app.reindexProjectHandler)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_diagnostics",
		Description: "Run diagnostic checks on a Go project. Accepts an optional list of categories (build, outdated, security, modernize); defaults to all categories.",
	}, app.getDiagnosticsHandler)

	return s
}

// toolError returns a CallToolResult with IsError=true.
func toolError(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, args...)},
		},
	}
}

// toolJSON returns a CallToolResult with a message and JSON-encoded data.
func toolJSON(msg string, data any) *mcp.CallToolResult {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return toolError("json marshal: %v", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg + "\n" + string(b)},
		},
	}
}
