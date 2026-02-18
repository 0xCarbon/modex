package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/canesin/modex/internal/server"
)

func main() {
	transport := flag.String("transport", "stdio", "transport to use: stdio or http")
	addr := flag.String("addr", "127.0.0.1:3838", "address to listen on (http transport only)")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	s := server.New()

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
