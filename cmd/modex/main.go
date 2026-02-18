package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xCarbon/modex/internal/db"
	"github.com/0xCarbon/modex/internal/server"
)

func main() {
	transport := flag.String("transport", "stdio", "transport to use: stdio or http")
	addr := flag.String("addr", "127.0.0.1:3838", "address to listen on (http transport only)")
	dbPath := flag.String("db", defaultDBPath(), "path to SQLite database")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

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

	s := server.New(database)

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
