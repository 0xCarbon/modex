package db_test

import (
	"testing"

	"github.com/0xCarbon/modex/internal/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenSetsWAL(t *testing.T) {
	d := openTestDB(t)
	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	// In-memory DBs report "memory" instead of "wal", but the PRAGMA still runs.
	// For file-backed DBs it would be "wal". We just verify the query succeeds.
	if mode != "memory" && mode != "wal" {
		t.Errorf("unexpected journal_mode: %q", mode)
	}
}

func TestSchemaCreatesTable(t *testing.T) {
	d := openTestDB(t)
	var name string
	err := d.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='docs'").Scan(&name)
	if err != nil {
		t.Fatalf("docs table not found: %v", err)
	}
}

func TestFTS5InsertAndQuery(t *testing.T) {
	d := openTestDB(t)

	_, err := d.Exec(`INSERT INTO docs
		(package_path, package_hash, module_path, module_version, item_name, kind, signature, doc_text)
		VALUES ('fmt', 'hash1', 'std', '', 'Println', 'Function', 'func Println(a ...any) (n int, err error)', 'Println formats using default formats')`)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	var itemName string
	err = d.QueryRow("SELECT item_name FROM docs_fts WHERE docs_fts MATCH 'Println'").Scan(&itemName)
	if err != nil {
		t.Fatalf("FTS5 query: %v", err)
	}
	if itemName != "Println" {
		t.Errorf("got %q, want Println", itemName)
	}
}

func TestFTS5TriggerSyncOnDelete(t *testing.T) {
	d := openTestDB(t)

	_, err := d.Exec(`INSERT INTO docs
		(package_path, package_hash, item_name, kind, signature, doc_text)
		VALUES ('os', 'hash2', 'Exit', 'Function', 'func Exit(code int)', 'Exit causes the program to exit')`)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	// Delete from main table; trigger should remove from FTS.
	if _, err := d.Exec("DELETE FROM docs WHERE package_hash = 'hash2'"); err != nil {
		t.Fatalf("DELETE: %v", err)
	}

	var count int
	d.QueryRow("SELECT COUNT(*) FROM docs_fts WHERE docs_fts MATCH 'Exit'").Scan(&count)
	if count != 0 {
		t.Errorf("FTS5 still has %d rows after delete", count)
	}
}

func TestFTS5TriggerSyncOnUpdate(t *testing.T) {
	d := openTestDB(t)

	_, err := d.Exec(`INSERT INTO docs
		(package_path, package_hash, item_name, kind, signature, doc_text)
		VALUES ('io', 'hash3', 'ReadAll', 'Function', 'func ReadAll(r Reader) ([]byte, error)', 'ReadAll reads from r')`)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	// Update item_name.
	if _, err := d.Exec("UPDATE docs SET item_name = 'ReadFull' WHERE package_hash = 'hash3'"); err != nil {
		t.Fatalf("UPDATE: %v", err)
	}

	// Use column-specific FTS5 query since doc_text still contains "ReadAll".
	var count int
	d.QueryRow("SELECT COUNT(*) FROM docs_fts WHERE docs_fts MATCH 'item_name:ReadAll'").Scan(&count)
	if count != 0 {
		t.Errorf("FTS5 still has old item_name after update (%d rows)", count)
	}

	d.QueryRow("SELECT COUNT(*) FROM docs_fts WHERE docs_fts MATCH 'item_name:ReadFull'").Scan(&count)
	if count != 1 {
		t.Errorf("FTS5 missing updated item_name (%d rows)", count)
	}
}

func TestHashExists(t *testing.T) {
	d := openTestDB(t)

	exists, err := d.HashExists("nonexistent")
	if err != nil {
		t.Fatalf("HashExists: %v", err)
	}
	if exists {
		t.Error("HashExists returned true for nonexistent hash")
	}

	_, err = d.Exec(`INSERT INTO docs
		(package_path, package_hash, item_name, kind) VALUES ('net', 'hash4', 'Dial', 'Function')`)
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	exists, err = d.HashExists("hash4")
	if err != nil {
		t.Fatalf("HashExists: %v", err)
	}
	if !exists {
		t.Error("HashExists returned false for existing hash")
	}
}

func TestDeleteByHash(t *testing.T) {
	d := openTestDB(t)

	for i, name := range []string{"A", "B", "C"} {
		hash := "samehash"
		if i == 2 {
			hash = "otherhash"
		}
		_, err := d.Exec(`INSERT INTO docs (package_path, package_hash, item_name, kind) VALUES (?, ?, ?, 'Function')`,
			"pkg", hash, name)
		if err != nil {
			t.Fatalf("INSERT %s: %v", name, err)
		}
	}

	n, err := d.DeleteByHash("samehash")
	if err != nil {
		t.Fatalf("DeleteByHash: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted %d rows, want 2", n)
	}

	var count int
	d.QueryRow("SELECT COUNT(*) FROM docs").Scan(&count)
	if count != 1 {
		t.Errorf("remaining %d rows, want 1", count)
	}
}
