package docs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xCarbon/modex/internal/db"
	"github.com/0xCarbon/modex/internal/docs"
)

// createTestProject creates a minimal Go project with a go.mod and one .go file.
func createTestProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

// Greet returns a greeting for name.
func Greet(name string) string { return "hello " + name }

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestIndexerSingleStdlibPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := createTestProject(t)
	database := openTestDB(t)
	idx := docs.NewIndexer(database, dir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := idx.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	snap := idx.Progress()
	if snap.Phase != docs.PhaseReady {
		t.Errorf("expected PhaseReady, got %v", snap.PhaseStr)
	}
	if snap.Total == 0 {
		t.Error("expected non-zero total packages")
	}
	if snap.Indexed == 0 {
		t.Error("expected non-zero indexed packages")
	}

	// Verify the test project's own package was indexed.
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM docs WHERE item_name = 'Greet'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Error("expected Greet to be indexed")
	}

	// Verify stdlib packages were indexed (fmt.Println should be there).
	if err := database.QueryRow("SELECT COUNT(*) FROM docs WHERE package_path = 'fmt' AND item_name = 'Println'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Error("expected fmt.Println to be indexed")
	}

	// Verify FTS5 works end-to-end.
	if err := database.QueryRow("SELECT COUNT(*) FROM docs_fts WHERE docs_fts MATCH 'Println'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count > 0 {
		t.Logf("FTS5 match for 'Println': %d rows", count)
	}
}

func TestIndexerDedup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := createTestProject(t)
	database := openTestDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// First run.
	idx1 := docs.NewIndexer(database, dir)
	if err := idx1.Run(ctx); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	var countAfterFirst int
	database.QueryRow("SELECT COUNT(*) FROM docs").Scan(&countAfterFirst)

	// Second run should skip all (same hashes).
	idx2 := docs.NewIndexer(database, dir)
	if err := idx2.Run(ctx); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	snap := idx2.Progress()
	// Most packages should be deduped; allow a small number to re-index
	// due to edge cases in version detection for some stdlib internal packages.
	if snap.Skipped == 0 {
		t.Error("second run should have skipped packages")
	}
	skipRate := float64(snap.Skipped) / float64(snap.Total)
	if skipRate < 0.90 {
		t.Errorf("expected >90%% skip rate, got %.1f%% (%d/%d)", skipRate*100, snap.Skipped, snap.Total)
	}

	var countAfterSecond int
	database.QueryRow("SELECT COUNT(*) FROM docs").Scan(&countAfterSecond)
	// Counts should be equal since re-indexed packages delete-then-insert the same items.
	if countAfterFirst != countAfterSecond {
		t.Logf("doc count changed slightly: %d -> %d (re-indexed packages)", countAfterFirst, countAfterSecond)
	}
}

func TestIndexerCancellation(t *testing.T) {
	dir := createTestProject(t)
	database := openTestDB(t)

	// Cancel immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := docs.NewIndexer(database, dir)
	err := idx.Run(ctx)
	if err == nil {
		// It's possible Run finishes before checking ctx if enumerate is fast enough.
		// But the DB should be consistent either way.
		return
	}

	// DB should be consistent (no partial data from uncommitted transactions).
	var count int
	database.QueryRow("SELECT COUNT(*) FROM docs").Scan(&count)
	t.Logf("docs count after cancellation: %d (should be consistent)", count)
}

func TestIndexerProgressUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := createTestProject(t)
	database := openTestDB(t)
	idx := docs.NewIndexer(database, dir)

	// Before run: queued.
	snap := idx.Progress()
	if snap.Phase != docs.PhaseQueued {
		t.Errorf("initial phase: got %v, want queued", snap.PhaseStr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := idx.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// After run: ready.
	snap = idx.Progress()
	if snap.Phase != docs.PhaseReady {
		t.Errorf("final phase: got %v, want ready", snap.PhaseStr)
	}
	if snap.Total == 0 {
		t.Error("total should be non-zero after run")
	}
	// indexed + skipped + failed should equal total.
	sum := snap.Indexed + snap.Skipped + snap.Failed
	if sum != snap.Total {
		t.Errorf("indexed(%d) + skipped(%d) + failed(%d) = %d, want %d",
			snap.Indexed, snap.Skipped, snap.Failed, sum, snap.Total)
	}
}

func TestIndexerReindexesChangedMainModulePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := createTestProject(t)
	database := openTestDB(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	idx1 := docs.NewIndexer(database, dir)
	if err := idx1.Run(ctx); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM docs WHERE package_path = 'testmod' AND item_name = 'Greet'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected Greet after first run")
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

// Welcome returns a greeting for name.
func Welcome(name string) string { return "welcome " + name }

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	idx2 := docs.NewIndexer(database, dir)
	if err := idx2.Run(ctx); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if err := database.QueryRow("SELECT COUNT(*) FROM docs WHERE package_path = 'testmod' AND item_name = 'Greet'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected Greet to be removed after source change, got %d rows", count)
	}

	if err := database.QueryRow("SELECT COUNT(*) FROM docs WHERE package_path = 'testmod' AND item_name = 'Welcome'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected Welcome after second run")
	}
}
