package server_test

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/canesin/modex/internal/server"
)

func TestNewReturnsServer(t *testing.T) {
	s := server.New()
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func connect(t *testing.T, s *mcp.Server) (*mcp.ClientSession, func()) {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	return cs, func() { cs.Close() }
}

func TestPingToolRegistered(t *testing.T) {
	cs, cleanup := connect(t, server.New())
	defer cleanup()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, tool := range res.Tools {
		if tool.Name == "ping" {
			return
		}
	}
	t.Errorf("ping tool not found in %v", res.Tools)
}

func TestPingToolExecutes(t *testing.T) {
	cs, cleanup := connect(t, server.New())
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ping",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("ping tool returned error: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("ping tool returned no content")
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *mcp.TextContent, got %T", res.Content[0])
	}
	if text.Text == "" {
		t.Fatal("ping tool returned empty text")
	}
}

func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := server.New()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	cs, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer cs.Close()

	cancel()

	if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Run returned unexpected error: %v", err)
	}
}
