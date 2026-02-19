package diagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

// govulncheckMessage is the subset of govulncheck -json output we care about.
// govulncheck streams one JSON object per line with a "type" discriminator.
type govulncheckMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type vulnFinding struct {
	OSV   string `json:"osv"`
	Trace []struct {
		Module   string `json:"module"`
		Package  string `json:"package"`
		Function string `json:"function"`
		Position string `json:"position"`
	} `json:"trace"`
}

func (o *Orchestrator) runSecurity(ctx context.Context) ([]Diagnostic, error) {
	out, err := runTool(ctx, o.ProjectPath, "govulncheck", "-json", "./...")
	if err != nil {
		// govulncheck not installed — skip silently.
		var pathErr *exec.Error
		if errors.As(err, &pathErr) {
			return nil, nil
		}
		return nil, err
	}

	var diags []Diagnostic
	dec := json.NewDecoder(strings.NewReader(out))
	for dec.More() {
		var msg govulncheckMessage
		if err := dec.Decode(&msg); err != nil {
			continue
		}
		if msg.Type != "Finding" {
			continue
		}
		var finding vulnFinding
		if err := json.Unmarshal(msg.Data, &finding); err != nil {
			continue
		}

		file := ""
		if len(finding.Trace) > 0 {
			file = finding.Trace[0].Position
		}
		diags = append(diags, Diagnostic{
			Category: CategorySecurity,
			File:     file,
			Message:  finding.OSV,
			Severity: SeverityError,
		})
	}
	return diags, nil
}
