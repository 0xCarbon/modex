// Package docs provides Go documentation extraction and background indexing.
package docs

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/format"
	"go/token"
	"os"
	"sort"
)

// DocItem represents one documentable symbol extracted from a Go package.
type DocItem struct {
	PackagePath   string
	PackageHash   string
	ModulePath    string
	ModuleVersion string
	GoVersion     string
	ItemName      string
	Kind          string // "Package", "Function", "Type", "Method", "Const", "Var"
	ParentName    string // for methods: the receiver type name
	ParentKind    string // "Struct", "Interface", or "" for top-level types
	Signature     string
	DocText       string
}

// PackageMeta holds module-level metadata attached to every DocItem from a package.
type PackageMeta struct {
	PackagePath   string
	PackageHash   string
	ModulePath    string
	ModuleVersion string
	GoVersion     string
}

// PackageHash computes a SHA256-based dedup key from package path, module version,
// and an optional content version.
// The package path is used (not module path) because a single Go module can contain
// many packages, and each needs independent dedup tracking.
func PackageHash(packagePath, moduleVersion, contentVersion string) string {
	h := sha256.Sum256([]byte(packagePath + "@" + moduleVersion + "#" + contentVersion))
	return fmt.Sprintf("%x", h[:16]) // 32-char hex, sufficient for dedup
}

// PackageContentVersion returns a stable content fingerprint for a package's files.
func PackageContentVersion(filePaths []string) (string, error) {
	sorted := append([]string(nil), filePaths...)
	sort.Strings(sorted)

	h := sha256.New()
	for _, path := range sorted {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		if _, err := h.Write([]byte(path)); err != nil {
			return "", fmt.Errorf("hash path %s: %w", path, err)
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash separator: %w", err)
		}
		if _, err := h.Write(b); err != nil {
			return "", fmt.Errorf("hash bytes %s: %w", path, err)
		}
		if _, err := h.Write([]byte{0}); err != nil {
			return "", fmt.Errorf("hash separator: %w", err)
		}
	}

	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum[:16]), nil
}

// ExtractFromDoc extracts all documentable symbols from a doc.Package.
func ExtractFromDoc(fset *token.FileSet, dpkg *doc.Package, meta PackageMeta) []DocItem {
	hash := meta.PackageHash
	if hash == "" {
		hash = PackageHash(meta.PackagePath, meta.ModuleVersion, "")
	}
	var items []DocItem

	base := DocItem{
		PackagePath:   meta.PackagePath,
		PackageHash:   hash,
		ModulePath:    meta.ModulePath,
		ModuleVersion: meta.ModuleVersion,
		GoVersion:     meta.GoVersion,
	}

	// Package-level doc.
	if dpkg.Doc != "" {
		item := base
		item.ItemName = dpkg.Name
		item.Kind = "Package"
		item.DocText = renderDoc(dpkg.Doc)
		items = append(items, item)
	}

	// Functions.
	for _, f := range dpkg.Funcs {
		item := base
		item.ItemName = f.Name
		item.Kind = "Function"
		item.Signature = renderDecl(fset, f.Decl)
		item.DocText = renderDoc(f.Doc)
		items = append(items, item)
	}

	// Constants.
	for _, v := range dpkg.Consts {
		for _, name := range v.Names {
			item := base
			item.ItemName = name
			item.Kind = "Const"
			item.Signature = renderDecl(fset, v.Decl)
			item.DocText = renderDoc(v.Doc)
			items = append(items, item)
		}
	}

	// Variables.
	for _, v := range dpkg.Vars {
		for _, name := range v.Names {
			item := base
			item.ItemName = name
			item.Kind = "Var"
			item.Signature = renderDecl(fset, v.Decl)
			item.DocText = renderDoc(v.Doc)
			items = append(items, item)
		}
	}

	// Types and their methods/funcs/consts/vars.
	for _, t := range dpkg.Types {
		parentKind := detectTypeKind(t)

		// The type itself.
		typeItem := base
		typeItem.ItemName = t.Name
		typeItem.Kind = "Type"
		typeItem.ParentKind = parentKind
		typeItem.Signature = renderDecl(fset, t.Decl)
		typeItem.DocText = renderDoc(t.Doc)
		items = append(items, typeItem)

		// Constructor-like functions (e.g. NewFoo).
		for _, f := range t.Funcs {
			item := base
			item.ItemName = f.Name
			item.Kind = "Function"
			item.ParentName = t.Name
			item.ParentKind = parentKind
			item.Signature = renderDecl(fset, f.Decl)
			item.DocText = renderDoc(f.Doc)
			items = append(items, item)
		}

		// Methods.
		for _, m := range t.Methods {
			item := base
			item.ItemName = m.Name
			item.Kind = "Method"
			item.ParentName = t.Name
			item.ParentKind = parentKind
			item.Signature = renderDecl(fset, m.Decl)
			item.DocText = renderDoc(m.Doc)
			items = append(items, item)
		}

		// Type-associated consts.
		for _, v := range t.Consts {
			for _, name := range v.Names {
				item := base
				item.ItemName = name
				item.Kind = "Const"
				item.ParentName = t.Name
				item.ParentKind = parentKind
				item.Signature = renderDecl(fset, v.Decl)
				item.DocText = renderDoc(v.Doc)
				items = append(items, item)
			}
		}

		// Type-associated vars.
		for _, v := range t.Vars {
			for _, name := range v.Names {
				item := base
				item.ItemName = name
				item.Kind = "Var"
				item.ParentName = t.Name
				item.ParentKind = parentKind
				item.Signature = renderDecl(fset, v.Decl)
				item.DocText = renderDoc(v.Doc)
				items = append(items, item)
			}
		}
	}

	return items
}

// detectTypeKind returns "Struct", "Interface", or "" based on the ast.TypeSpec.
func detectTypeKind(t *doc.Type) string {
	if t.Decl == nil || len(t.Decl.Specs) == 0 {
		return ""
	}
	ts, ok := t.Decl.Specs[0].(*ast.TypeSpec)
	if !ok {
		return ""
	}
	switch ts.Type.(type) {
	case *ast.StructType:
		return "Struct"
	case *ast.InterfaceType:
		return "Interface"
	default:
		return ""
	}
}

// renderDecl formats an AST declaration to a compact signature string.
// For function declarations, the body is stripped.
func renderDecl(fset *token.FileSet, decl ast.Decl) string {
	if decl == nil {
		return ""
	}

	// Strip function bodies for cleaner signatures.
	if fd, ok := decl.(*ast.FuncDecl); ok {
		clone := *fd
		clone.Body = nil
		decl = &clone
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, decl); err != nil {
		return ""
	}
	return buf.String()
}

// renderDoc converts raw doc comment text to clean plaintext using go/doc/comment.
func renderDoc(raw string) string {
	if raw == "" {
		return ""
	}
	var p comment.Parser
	d := p.Parse(raw)
	var pr comment.Printer
	return string(bytes.TrimSpace(pr.Text(d)))
}
