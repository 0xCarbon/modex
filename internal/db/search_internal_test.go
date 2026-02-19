package db

import "testing"

func TestSymbolQueryUsesLastDotForImportPaths(t *testing.T) {
	got := symbolQuery("github.com/org/pkg.Func")
	want := "item_name:Func* AND package_path:github.com/org/pkg*"
	if got != want {
		t.Fatalf("symbolQuery = %q, want %q", got, want)
	}
}

func TestSymbolQueryParsesMethodPattern(t *testing.T) {
	got := symbolQuery("bytes.Buffer.Write")
	want := "item_name:Write* AND parent_name:Buffer* AND package_path:bytes*"
	if got != want {
		t.Fatalf("symbolQuery = %q, want %q", got, want)
	}
}

func TestSymbolQueryParsesImportPathMethodPattern(t *testing.T) {
	got := symbolQuery("github.com/org/pkg.Type.Method")
	want := "item_name:Method* AND parent_name:Type* AND package_path:github.com/org/pkg*"
	if got != want {
		t.Fatalf("symbolQuery = %q, want %q", got, want)
	}
}
