package diagnostics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/0xCarbon/modex/internal/diagnostics"
)

func TestRunAllCategories(t *testing.T) {
	o := &diagnostics.Orchestrator{ProjectPath: t.TempDir()}
	got, err := o.Run(context.Background(), diagnostics.AllCategories)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got == nil {
		t.Fatal("Run returned nil slice")
	}
}

func TestRunSubsetCategories(t *testing.T) {
	o := &diagnostics.Orchestrator{ProjectPath: t.TempDir()}
	got, err := o.Run(context.Background(), []diagnostics.Category{diagnostics.CategoryBuild})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got == nil {
		t.Fatal("Run returned nil slice")
	}
}

func TestRunNilCategories(t *testing.T) {
	o := &diagnostics.Orchestrator{ProjectPath: t.TempDir()}
	got, err := o.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got == nil {
		t.Fatal("Run returned nil slice")
	}
}

func TestRunCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	o := &diagnostics.Orchestrator{ProjectPath: t.TempDir()}
	_, err := o.Run(ctx, diagnostics.AllCategories)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
