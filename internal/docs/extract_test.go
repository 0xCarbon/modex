package docs_test

import (
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"testing"

	"github.com/0xCarbon/modex/internal/docs"
)

// parsePkg is a test helper that parses Go source and returns a doc.Package.
func parsePkg(t *testing.T, src string) (*token.FileSet, *doc.Package) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	dpkg := doc.New(&ast.Package{
		Name:  f.Name.Name,
		Files: map[string]*ast.File{"test.go": f},
	}, "example.com/test", doc.AllDecls)
	return fset, dpkg
}

func meta() docs.PackageMeta {
	return docs.PackageMeta{
		PackagePath:   "example.com/test",
		ModulePath:    "example.com/test",
		ModuleVersion: "v1.0.0",
		GoVersion:     "1.25",
	}
}

func findItem(items []docs.DocItem, name, kind string) *docs.DocItem {
	for i := range items {
		if items[i].ItemName == name && items[i].Kind == kind {
			return &items[i]
		}
	}
	return nil
}

func TestExtractFunction(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// Add returns the sum of a and b.
func Add(a, b int) int { return a + b }
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	item := findItem(items, "Add", "Function")
	if item == nil {
		t.Fatal("Function Add not found")
	}
	if item.Signature == "" {
		t.Error("expected non-empty signature")
	}
	if item.DocText == "" {
		t.Error("expected non-empty doc text")
	}
	if item.ParentName != "" {
		t.Errorf("expected empty parent, got %q", item.ParentName)
	}
}

func TestExtractType(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// Widget is a UI component.
type Widget struct {
	Name string
}

// NewWidget creates a Widget.
func NewWidget(name string) *Widget { return &Widget{Name: name} }

// Render draws the widget.
func (w *Widget) Render() string { return w.Name }
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())

	typ := findItem(items, "Widget", "Type")
	if typ == nil {
		t.Fatal("Type Widget not found")
	}
	if typ.ParentKind != "Struct" {
		t.Errorf("expected ParentKind=Struct, got %q", typ.ParentKind)
	}

	ctor := findItem(items, "NewWidget", "Function")
	if ctor == nil {
		t.Fatal("Function NewWidget not found")
	}
	if ctor.ParentName != "Widget" {
		t.Errorf("expected ParentName=Widget, got %q", ctor.ParentName)
	}

	meth := findItem(items, "Render", "Method")
	if meth == nil {
		t.Fatal("Method Render not found")
	}
	if meth.ParentName != "Widget" {
		t.Errorf("expected ParentName=Widget, got %q", meth.ParentName)
	}
	if meth.ParentKind != "Struct" {
		t.Errorf("expected ParentKind=Struct, got %q", meth.ParentKind)
	}
}

func TestExtractInterface(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// Reader reads bytes.
type Reader interface {
	Read(p []byte) (n int, err error)
}
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	typ := findItem(items, "Reader", "Type")
	if typ == nil {
		t.Fatal("Type Reader not found")
	}
	if typ.ParentKind != "Interface" {
		t.Errorf("expected ParentKind=Interface, got %q", typ.ParentKind)
	}
}

func TestExtractConst(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// MaxSize is the maximum allowed size.
const MaxSize = 1024
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	item := findItem(items, "MaxSize", "Const")
	if item == nil {
		t.Fatal("Const MaxSize not found")
	}
}

func TestExtractVar(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// ErrNotFound is returned when an item is not found.
var ErrNotFound = errors.New("not found")
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	item := findItem(items, "ErrNotFound", "Var")
	if item == nil {
		t.Fatal("Var ErrNotFound not found")
	}
}

func TestPackageHash(t *testing.T) {
	h1 := docs.PackageHash("example.com/foo", "v1.0.0", "")
	h2 := docs.PackageHash("example.com/foo", "v1.0.0", "")
	h3 := docs.PackageHash("example.com/foo", "v1.1.0", "")
	h4 := docs.PackageHash("example.com/foo", "", "content-v1")
	h5 := docs.PackageHash("example.com/foo", "", "content-v2")

	if h1 != h2 {
		t.Error("same inputs should produce same hash")
	}
	if h1 == h3 {
		t.Error("different versions should produce different hashes")
	}
	if h4 == h5 {
		t.Error("different content versions should produce different hashes")
	}
	if len(h1) != 32 {
		t.Errorf("expected 32-char hex hash, got %d chars", len(h1))
	}
}

func TestExtractPackageDoc(t *testing.T) {
	fset, dpkg := parsePkg(t, `
// Package test provides testing utilities.
package test
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	item := findItem(items, "test", "Package")
	if item == nil {
		t.Fatal("Package doc not found")
	}
	if item.DocText == "" {
		t.Error("expected non-empty package doc")
	}
}

func TestSignatureStripsBody(t *testing.T) {
	fset, dpkg := parsePkg(t, `
package test

// LongFunc does something complex.
func LongFunc(x int) int {
	for i := 0; i < x; i++ {
		_ = i * 2
	}
	return x
}
`)
	items := docs.ExtractFromDoc(fset, dpkg, meta())
	item := findItem(items, "LongFunc", "Function")
	if item == nil {
		t.Fatal("Function LongFunc not found")
	}
	// Signature should not contain the body.
	if contains(item.Signature, "for") || contains(item.Signature, "return") {
		t.Errorf("signature should not contain body: %s", item.Signature)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchString(s, sub)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
