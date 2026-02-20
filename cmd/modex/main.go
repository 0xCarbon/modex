package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xCarbon/modex/internal/db"
	"github.com/0xCarbon/modex/internal/logbuf"
	"github.com/0xCarbon/modex/internal/server"
)

// version is set by goreleaser ldflags: -X main.version={{ .Version }}
var version = "dev"

// Default values shared between main.go and service.go.
const (
	DefaultRateLimitPerSecond = 30
	DefaultMaxConcurrent      = 64
)

func main() {
	// Determine subcommand. Default is "server".
	sub := "server"
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}

	switch sub {
	case "server":
		cmdServer(args)
	case "version":
		fmt.Printf("modex %s\n", version)
	case "setup":
		cmdSetup()
	case "update":
		cmdUpdate(args)
	case "install-service":
		cmdInstallService(args)
	case "remove-service":
		cmdRemoveService(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", sub)
		fmt.Fprintf(os.Stderr, "usage: modex [server|version|setup|update|install-service|remove-service]\n")
		os.Exit(1)
	}
}

func cmdServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	transport := fs.String("transport", "http", "transport: http or stdio")
	addr := fs.String("addr", "127.0.0.1:3838", "address to listen on (http transport only)")
	dbPath := fs.String("db", defaultDBPath(), "path to SQLite database")
	rateLimit := fs.Int("rate-limit", DefaultRateLimitPerSecond, "max requests per second")
	maxConcurrent := fs.Int("max-concurrent", DefaultMaxConcurrent, "max concurrent requests")
	fs.Parse(args)

	// Set up log buffer + handler.
	logBuf := logbuf.NewRingBuffer(500)
	slog.SetDefault(slog.New(logbuf.NewHandler(logBuf)))

	// Warn on non-loopback binding.
	if *transport == "http" {
		host, _, err := net.SplitHostPort(*addr)
		if err == nil && !isLoopback(host) {
			slog.Warn("binding to non-loopback address without authentication", "addr", *addr)
		}
	}

	// Ensure DB directory exists.
	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		slog.Error("failed to create db directory", "err", err)
		os.Exit(1)
	}

	database, err := db.Open(*dbPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	s := server.New(database, server.Config{
		RateLimitPerSecond: *rateLimit,
		MaxConcurrent:      *maxConcurrent,
		LogBuffer:          logBuf,
	})

	switch *transport {
	case "stdio":
		runStdio(s)
	case "http":
		runHTTP(s, *addr)
	default:
		fmt.Fprintf(os.Stderr, "unknown transport %q; use stdio or http\n", *transport)
		os.Exit(1)
	}
}

func defaultDBPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	return filepath.Join(cacheDir, "modex", "modex.db")
}

func runStdio(s *mcp.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := s.Run(ctx, &mcp.StdioTransport{}); err != nil && err != ctx.Err() {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

func runHTTP(s *mcp.Server, addr string) {
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s
	}, nil)

	httpSrv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", addr, "transport", "http")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
