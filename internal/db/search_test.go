package db_test

import (
	"testing"

	"github.com/0xCarbon/modex/internal/db"
)

func TestBuildFTSQueryModes(t *testing.T) {
	tests := []struct {
		query, mode string
		wantContain string
	}{
		{"Println", "text", "Println*"},
		{"fmt.Println", "symbol", "item_name:Println*"},
		{"fmt.Println", "auto", "item_name:Println*"},
		{"Println", "auto", "Println*"},
		{"raw AND query", "fts5", "raw AND query"},
	}
	for _, tt := range tests {
		t.Run(tt.mode+"/"+tt.query, func(t *testing.T) {
			// buildFTSQuery is unexported; test via SearchDocs returning no error.
			d, err := db.Open(":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			resp, err := d.SearchDocs(db.SearchParams{Query: tt.query, Mode: tt.mode})
			if err != nil {
				t.Fatalf("SearchDocs: %v", err)
			}
			// Empty DB returns empty results, but no error means query was well-formed.
			if resp.Results == nil {
				// nil vs empty slice — normalise
				resp.Results = []db.SearchResult{}
			}
		})
	}
}

func TestSearchDocsInsertAndFind(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// Insert a synthetic doc row via the underlying sql.DB.
	_, err = d.Exec(`INSERT INTO docs
		(package_path, package_hash, module_path, module_version, go_version,
		 item_name, kind, parent_name, parent_kind, signature, doc_text)
		VALUES (?,?,?,?,?, ?,?,?,?,?,?)`,
		"fmt", "hash1", "fmt", "go1.26", "1.26",
		"Println", "func", "", "", "func Println(a ...any) (n int, err error)",
		"Println formats using the default formats for its operands.")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := d.SearchDocs(db.SearchParams{Query: "Println", Mode: "text"})
	if err != nil {
		t.Fatalf("SearchDocs: %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].ItemName != "Println" {
		t.Errorf("ItemName = %q", resp.Results[0].ItemName)
	}
}

func TestSearchDocsFilterByKind(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	for _, row := range []struct{ name, kind string }{
		{"Reader", "interface"},
		{"Read", "func"},
	} {
		_, err = d.Exec(`INSERT INTO docs
			(package_path, package_hash, module_path, module_version, go_version,
			 item_name, kind, parent_name, parent_kind, signature, doc_text)
			VALUES (?,?,?,?,?, ?,?,?,?,?,?)`,
			"io", "hash2", "io", "go1.26", "1.26",
			row.name, row.kind, "", "", row.name+" sig", row.name+" doc")
		if err != nil {
			t.Fatal(err)
		}
	}

	resp, err := d.SearchDocs(db.SearchParams{Query: "Read", Mode: "text", Kinds: []string{"interface"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range resp.Results {
		if r.Kind != "interface" {
			t.Errorf("got kind %q, want interface", r.Kind)
		}
	}
}

func TestSearchDocsPagination(t *testing.T) {
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// Insert 12 rows all matching "test".
	for i := range 12 {
		_, err = d.Exec(`INSERT INTO docs
			(package_path, package_hash, module_path, module_version, go_version,
			 item_name, kind, signature, doc_text)
			VALUES (?,?,?,?,?, ?,?,?,?)`,
			"pkg", "hash3", "pkg", "v1.0.0", "1.26",
			"TestFunc", "func", "func TestFunc()", "test function doc number")
		if err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	resp, err := d.SearchDocs(db.SearchParams{Query: "TestFunc", Mode: "text", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 10 {
		t.Errorf("got %d results, want 10", len(resp.Results))
	}
	if !resp.HasMore {
		t.Error("expected HasMore=true")
	}
}
