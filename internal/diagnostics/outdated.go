package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// moduleInfo is the relevant subset of `go list -m -json` output.
type moduleInfo struct {
	Path    string      `json:"Path"`
	Version string      `json:"Version"`
	Update  *moduleInfo `json:"Update,omitempty"`
}

func (o *Orchestrator) runOutdated(ctx context.Context) ([]Diagnostic, error) {
	out, err := runTool(ctx, o.ProjectPath, "go", "list", "-m", "-u", "-json", "all")
	if err != nil {
		return nil, err
	}
	return parseOutdatedJSON(out), nil
}

func parseOutdatedJSON(output string) []Diagnostic {
	var diags []Diagnostic
	dec := json.NewDecoder(strings.NewReader(output))
	for dec.More() {
		var mod moduleInfo
		if err := dec.Decode(&mod); err != nil {
			continue
		}
		if mod.Update == nil {
			continue
		}
		diags = append(diags, Diagnostic{
			Category: CategoryOutdated,
			File:     "go.mod",
			Message: fmt.Sprintf("%s: %s -> %s",
				mod.Path, mod.Version, mod.Update.Version),
			Severity: SeverityInfo,
		})
	}
	return diags
}
