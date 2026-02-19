package docs

import (
	"context"
	"database/sql"
	"fmt"
	"go/ast"
	"go/doc"
	"log/slog"
	"sync"

	"golang.org/x/tools/go/packages"

	"github.com/0xCarbon/modex/internal/db"
)

// Phase represents the current indexing phase.
type Phase int

const (
	PhaseQueued   Phase = iota // Waiting to start
	PhaseLoading               // Enumerating packages
	PhaseIndexing              // Extracting and inserting docs
	PhaseReady                 // Indexing complete
	PhaseFailed                // Indexing failed
)

func (p Phase) String() string {
	switch p {
	case PhaseQueued:
		return "queued"
	case PhaseLoading:
		return "loading"
	case PhaseIndexing:
		return "indexing"
	case PhaseReady:
		return "ready"
	case PhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ProgressSnapshot is a point-in-time view of indexing progress.
type ProgressSnapshot struct {
	Phase       Phase  `json:"phase"`
	PhaseStr    string `json:"phase_str"`
	Total       int    `json:"total"`
	Indexed     int    `json:"indexed"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	LastError   string `json:"last_error,omitempty"`
	ProjectPath string `json:"project_path"`
}

// Progress tracks indexing progress with thread-safe access.
type Progress struct {
	mu          sync.Mutex
	phase       Phase
	total       int
	indexed     int
	skipped     int
	failed      int
	lastError   string
	projectPath string
}

// Snapshot returns a point-in-time copy of the progress state.
func (p *Progress) Snapshot() ProgressSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return ProgressSnapshot{
		Phase:       p.phase,
		PhaseStr:    p.phase.String(),
		Total:       p.total,
		Indexed:     p.indexed,
		Skipped:     p.skipped,
		Failed:      p.failed,
		LastError:   p.lastError,
		ProjectPath: p.projectPath,
	}
}

// Indexer orchestrates background documentation indexing for a Go project.
type Indexer struct {
	db         *db.DB
	progress   Progress
	projectDir string
}

// NewIndexer creates an indexer for the given project directory.
func NewIndexer(database *db.DB, projectDir string) *Indexer {
	return &Indexer{
		db:         database,
		projectDir: projectDir,
		progress:   Progress{projectPath: projectDir},
	}
}

// Progress returns a snapshot of the current indexing progress.
func (idx *Indexer) Progress() ProgressSnapshot {
	return idx.progress.Snapshot()
}

func (idx *Indexer) setFailed(msg string) {
	idx.progress.mu.Lock()
	idx.progress.phase = PhaseFailed
	idx.progress.lastError = msg
	idx.progress.mu.Unlock()
}

// Run executes the two-phase indexing pipeline:
// 1. Enumerate stdlib + project packages (lightweight, NeedName|NeedModule)
// 2. For each non-deduped package, full-load and extract docs
func (idx *Indexer) Run(ctx context.Context) error {
	idx.progress.mu.Lock()
	idx.progress.phase = PhaseLoading
	idx.progress.mu.Unlock()

	slog.Info("indexer: enumerating packages", "project", idx.projectDir)

	// Phase 1: Enumerate packages.
	pkgPaths, err := idx.enumerate(ctx)
	if err != nil {
		idx.setFailed(fmt.Sprintf("enumerate: %v", err))
		return fmt.Errorf("enumerate: %w", err)
	}

	idx.progress.mu.Lock()
	idx.progress.total = len(pkgPaths)
	idx.progress.phase = PhaseIndexing
	idx.progress.mu.Unlock()

	slog.Info("indexer: indexing packages", "total", len(pkgPaths), "project", idx.projectDir)

	// Phase 2: Index each package sequentially.
	for _, pp := range pkgPaths {
		if err := ctx.Err(); err != nil {
			idx.setFailed(fmt.Sprintf("cancelled: %v", err))
			return err
		}
		if err := idx.indexPackage(ctx, pp); err != nil {
			slog.Warn("indexer: package failed", "pkg", pp.path, "err", err)
			idx.progress.mu.Lock()
			idx.progress.failed++
			idx.progress.lastError = fmt.Sprintf("%s: %v", pp.path, err)
			idx.progress.mu.Unlock()
		}
	}

	idx.progress.mu.Lock()
	idx.progress.phase = PhaseReady
	idx.progress.mu.Unlock()

	snap := idx.progress.Snapshot()
	slog.Info("indexer: done",
		"indexed", snap.Indexed, "skipped", snap.Skipped,
		"failed", snap.Failed, "total", snap.Total)
	return nil
}

type pkgInfo struct {
	path       string
	modulePath string
	moduleVer  string
	goVersion  string
}

// enumerate discovers stdlib + project dependency packages using lightweight loading.
func (idx *Indexer) enumerate(ctx context.Context) ([]pkgInfo, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode:    packages.NeedName | packages.NeedModule,
		Dir:     idx.projectDir,
	}

	// Load stdlib.
	stdPkgs, err := packages.Load(cfg, "std")
	if err != nil {
		return nil, fmt.Errorf("load std: %w", err)
	}

	// Load project dependencies.
	allPkgs, err := packages.Load(cfg, "all")
	if err != nil {
		return nil, fmt.Errorf("load all: %w", err)
	}

	seen := make(map[string]bool)
	var result []pkgInfo

	for _, pkg := range stdPkgs {
		if seen[pkg.PkgPath] || len(pkg.Errors) > 0 {
			continue
		}
		seen[pkg.PkgPath] = true
		result = append(result, pkgInfo{
			path:       pkg.PkgPath,
			modulePath: "std",
			moduleVer:  goVersion(pkg),
			goVersion:  goVersion(pkg),
		})
	}

	for _, pkg := range allPkgs {
		if seen[pkg.PkgPath] || len(pkg.Errors) > 0 {
			continue
		}
		seen[pkg.PkgPath] = true
		if pkg.Module == nil {
			continue // skip packages without module info
		}
		result = append(result, pkgInfo{
			path:       pkg.PkgPath,
			modulePath: pkg.Module.Path,
			moduleVer:  pkg.Module.Version,
			goVersion:  pkg.Module.GoVersion,
		})
	}

	return result, nil
}

func goVersion(pkg *packages.Package) string {
	if pkg.Module != nil && pkg.Module.GoVersion != "" {
		return pkg.Module.GoVersion
	}
	return ""
}

// indexPackage loads a single package with full syntax and indexes its docs.
func (idx *Indexer) indexPackage(ctx context.Context, pi pkgInfo) error {
	var (
		hash string
		pkg  *packages.Package
		err  error
	)

	// For local/replaced packages with empty module versions, include a package
	// content fingerprint so source edits invalidate dedup.
	if pi.moduleVer == "" {
		pkg, err = idx.loadPackage(ctx, pi.path)
		if err != nil {
			return err
		}
		contentVersion, err := PackageContentVersion(pkg.GoFiles)
		if err != nil {
			return fmt.Errorf("content version %s: %w", pi.path, err)
		}
		hash = PackageHash(pi.path, pi.moduleVer, contentVersion)
	} else {
		hash = PackageHash(pi.path, pi.moduleVer, "")
	}

	// Check dedup.
	exists, err := idx.db.HashExists(hash)
	if err != nil {
		return fmt.Errorf("dedup check: %w", err)
	}
	if exists {
		idx.progress.mu.Lock()
		idx.progress.skipped++
		idx.progress.mu.Unlock()
		return nil
	}

	// Full load with syntax.
	if pkg == nil {
		pkg, err = idx.loadPackage(ctx, pi.path)
		if err != nil {
			return err
		}
	}

	// Build go/doc.Package from syntax.
	fset := pkg.Fset
	files := make(map[string]*ast.File)
	for i, f := range pkg.Syntax {
		name := "file"
		if i < len(pkg.GoFiles) {
			name = pkg.GoFiles[i]
		}
		files[name] = f
	}
	dpkg := doc.New(&ast.Package{
		Name:  pkg.Name,
		Files: files,
	}, pi.path, doc.AllDecls)

	// Extract items.
	items := ExtractFromDoc(fset, dpkg, PackageMeta{
		PackagePath:   pi.path,
		PackageHash:   hash,
		ModulePath:    pi.modulePath,
		ModuleVersion: pi.moduleVer,
		GoVersion:     pi.goVersion,
	})

	// Insert in transaction: delete old hash, insert new items.
	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM docs WHERE package_path = ?", pi.path); err != nil {
		return fmt.Errorf("delete old: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO docs
		(package_path, package_hash, module_path, module_version, go_version,
		 item_name, kind, parent_name, parent_kind, signature, doc_text)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		var parentName, parentKind sql.NullString
		if item.ParentName != "" {
			parentName = sql.NullString{String: item.ParentName, Valid: true}
		}
		if item.ParentKind != "" {
			parentKind = sql.NullString{String: item.ParentKind, Valid: true}
		}
		if _, err := stmt.Exec(
			item.PackagePath, item.PackageHash, item.ModulePath, item.ModuleVersion, item.GoVersion,
			item.ItemName, item.Kind, parentName, parentKind, item.Signature, item.DocText,
		); err != nil {
			return fmt.Errorf("insert %s: %w", item.ItemName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	idx.progress.mu.Lock()
	idx.progress.indexed++
	idx.progress.mu.Unlock()

	return nil
}

func (idx *Indexer) loadPackage(ctx context.Context, packagePath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedModule,
		Dir:     idx.projectDir,
	}
	pkgs, err := packages.Load(cfg, packagePath)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", packagePath, err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %s", packagePath)
	}
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package errors: %v", pkg.Errors[0])
	}
	return pkg, nil
}
