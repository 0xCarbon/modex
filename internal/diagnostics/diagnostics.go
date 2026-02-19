package diagnostics

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Category identifies a diagnostic check type.
type Category string

const (
	CategoryBuild     Category = "build"
	CategoryOutdated  Category = "outdated"
	CategorySecurity  Category = "security"
	CategoryModernize Category = "modernize"
)

// AllCategories lists every supported diagnostic category.
var AllCategories = []Category{
	CategoryBuild,
	CategoryOutdated,
	CategorySecurity,
	CategoryModernize,
}

// Severity indicates how serious a diagnostic finding is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Diagnostic represents a single finding from a diagnostic check.
type Diagnostic struct {
	Category Category `json:"category"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

// Orchestrator runs diagnostic categories against a Go project.
type Orchestrator struct {
	ProjectPath string
}

// Run executes the requested categories in parallel and returns all findings.
// If categories is nil or empty, AllCategories is used.
// Returns context.Canceled if ctx is already done before any work starts.
func (o *Orchestrator) Run(ctx context.Context, categories []Category) ([]Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(categories) == 0 {
		categories = AllCategories
	}

	var (
		mu      sync.Mutex
		results []Diagnostic
	)

	g, ctx := errgroup.WithContext(ctx)
	for _, cat := range categories {
		cat := cat
		g.Go(func() error {
			var diags []Diagnostic
			var err error
			switch cat {
			case CategoryBuild:
				diags, err = o.runBuild(ctx)
			case CategoryOutdated:
				diags, err = o.runOutdated(ctx)
			case CategorySecurity:
				diags, err = o.runSecurity(ctx)
			case CategoryModernize:
				diags, err = o.runModernize(ctx)
			}
			if err != nil {
				return err
			}
			if len(diags) > 0 {
				mu.Lock()
				results = append(results, diags...)
				mu.Unlock()
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	if results == nil {
		results = []Diagnostic{}
	}
	return results, nil
}

// runBuild, runOutdated, runSecurity, and runModernize are implemented in
// build.go, outdated.go, security.go, and modernize.go respectively.
