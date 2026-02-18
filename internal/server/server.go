package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"
)

const (
	defaultRateLimitPerSecond   = 3 * 10
	defaultRateLimitBurst       = defaultRateLimitPerSecond
	defaultMaxConcurrentRequest = 1 << 6
)

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

// New returns a configured MCP server with rate limiting, concurrency limiting,
// and a placeholder ping tool.
func New() *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{Name: "modex", Version: "0.1.0"},
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
	return s
}
