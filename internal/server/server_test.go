package server_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/0xCarbon/modex/internal/db"
	"github.com/0xCarbon/modex/internal/server"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func newServer(t *testing.T) *mcp.Server {
	t.Helper()
	return server.New(openTestDB(t))
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

func TestNewReturnsServer(t *testing.T) {
	s := newServer(t)
	if s == nil {
		t.Fatal("New() returned nil")
	}
}

func TestPingToolRegistered(t *testing.T) {
	cs, cleanup := connect(t, newServer(t))
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
	cs, cleanup := connect(t, newServer(t))
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
	s := newServer(t)
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

func TestRegisterProjectTool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Create a minimal Go project.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "register_project",
		Arguments: map[string]any{"project_path": dir},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("register_project returned error: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("register_project returned no content")
	}
}

func TestRegisterProjectNoGoMod(t *testing.T) {
	dir := t.TempDir() // Empty directory, no go.mod.

	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "register_project",
		Arguments: map[string]any{"project_path": dir},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Error("expected error for missing go.mod")
	}
}

func TestRegisterProjectEmptyPath(t *testing.T) {
	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "register_project",
		Arguments: map[string]any{"project_path": ""},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error for empty project_path")
	}
}

func TestGetIndexStatusTool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644)

	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	// Register first.
	_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "register_project",
		Arguments: map[string]any{"project_path": dir},
	})
	if err != nil {
		t.Fatalf("register_project: %v", err)
	}

	// Get status.
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_index_status",
		Arguments: map[string]any{"project_path": dir},
	})
	if err != nil {
		t.Fatalf("get_index_status: %v", err)
	}
	if res.IsError {
		t.Fatalf("get_index_status returned error: %v", res.Content)
	}
}

func TestGetIndexStatusUnregistered(t *testing.T) {
	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "get_index_status",
		Arguments: map[string]any{"project_path": "/nonexistent"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Error("expected error for unregistered project")
	}
}

func TestToolsRegistered(t *testing.T) {
	cs, cleanup := connect(t, newServer(t))
	defer cleanup()

	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	want := map[string]bool{
		"ping":             false,
		"register_project": false,
		"get_index_status": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not registered", name)
		}
	}
}
