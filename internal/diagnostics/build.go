package diagnostics

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"
)

// diagLineRE matches "file.go:line:col: message" or "file.go:line: message".
var diagLineRE = regexp.MustCompile(`^(.+?):(\d+)(?::\d+)?:\s+(.+)$`)

// parseDiagLines converts plain-text compiler/vet output into Diagnostics.
func parseDiagLines(output string, category Category, severity Severity) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := diagLineRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNum, _ := strconv.Atoi(m[2])
		diags = append(diags, Diagnostic{
			Category: category,
			File:     m[1],
			Line:     lineNum,
			Message:  m[3],
			Severity: severity,
		})
	}
	return diags
}

func (o *Orchestrator) runBuild(ctx context.Context) ([]Diagnostic, error) {
	var diags []Diagnostic

	// go build reports compilation errors.
	out, err := runTool(ctx, o.ProjectPath, "go", "build", "./...")
	if err != nil {
		return nil, err
	}
	diags = append(diags, parseDiagLines(out, CategoryBuild, SeverityError)...)

	// go vet reports common code-correctness warnings.
	out, err = runTool(ctx, o.ProjectPath, "go", "vet", "./...")
	if err != nil {
		return nil, err
	}
	diags = append(diags, parseDiagLines(out, CategoryBuild, SeverityWarning)...)

	return diags, nil
}
