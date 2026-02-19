package diagnostics

import (
	"bufio"
	"context"
	"strconv"
	"strings"
)

// RunModernize detects modernization opportunities via go fix -diff without modifying files.
// Exported for use by the apply_modernize tool handler.
func RunModernize(ctx context.Context, projectPath string, fixers []string, aggressive bool) ([]Diagnostic, error) {
	o := &Orchestrator{ProjectPath: projectPath}
	args := buildFixArgs(fixers, aggressive, true)
	return o.runFixDiff(ctx, args)
}

// ApplyModernize applies go fix to the project, optionally with selective fixers.
// When dryRun is true it returns diffs as diagnostics without modifying files.
// It runs up to maxPasses times for iterative convergence.
func ApplyModernize(ctx context.Context, projectPath string, fixers []string, aggressive, dryRun bool) ([]Diagnostic, error) {
	o := &Orchestrator{ProjectPath: projectPath}
	const maxPasses = 3
	var all []Diagnostic
	for range maxPasses {
		args := buildFixArgs(fixers, aggressive, dryRun)
		diags, err := o.runFixDiff(ctx, args)
		if err != nil {
			return nil, err
		}
		all = append(all, diags...)
		if len(diags) == 0 {
			break // converged
		}
		if dryRun {
			break // dry-run: don't iterate
		}
	}
	return all, nil
}

func (o *Orchestrator) runModernize(ctx context.Context) ([]Diagnostic, error) {
	return o.runFixDiff(ctx, buildFixArgs(nil, false, true))
}

// buildFixArgs constructs the go fix argument list.
func buildFixArgs(fixers []string, aggressive, diff bool) []string {
	args := []string{"fix"}
	if diff {
		args = append(args, "-diff")
	}
	if len(fixers) > 0 {
		// Disable all fixers first, then enable requested ones.
		args = append(args, "-all=false")
		for _, f := range fixers {
			args = append(args, "-"+f+"=true")
		}
	} else if aggressive {
		// Enable all fixers including disabled-by-default ones.
		args = append(args, "-appendclipped=true", "-bloop=true", "-slicesdelete=true")
	}
	args = append(args, "./...")
	return args
}

// runFixDiff runs go fix and parses the unified diff output into Diagnostics.
func (o *Orchestrator) runFixDiff(ctx context.Context, args []string) ([]Diagnostic, error) {
	out, err := runTool(ctx, o.ProjectPath, "go", args...)
	if err != nil {
		return nil, err
	}
	return parseDiff(out), nil
}

// parseDiff converts a unified diff produced by go fix -diff into Diagnostics.
func parseDiff(diff string) []Diagnostic {
	var diags []Diagnostic
	var currentFile string
	var oldLine int

	scanner := bufio.NewScanner(strings.NewReader(diff))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "--- "):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentFile = strings.TrimPrefix(parts[1], "a/")
			}
		case strings.HasPrefix(line, "@@ "):
			oldLine = parseHunkStart(line)
		case strings.HasPrefix(line, "-") && currentFile != "":
			removed := strings.TrimPrefix(line, "-")
			diags = append(diags, Diagnostic{
				Category: CategoryModernize,
				File:     currentFile,
				Line:     oldLine,
				Message:  "modernize: " + strings.TrimSpace(removed),
				Severity: SeverityInfo,
			})
			oldLine++
		case strings.HasPrefix(line, "+"):
			// new line — don't increment oldLine
		case strings.HasPrefix(line, " "):
			oldLine++
		}
	}
	return diags
}

// parseHunkStart extracts the starting line number from a unified diff @@ header.
func parseHunkStart(line string) int {
	after, ok := strings.CutPrefix(line, "@@ -")
	if !ok {
		return 0
	}
	before, _, _ := strings.Cut(after, " ")
	numStr, _, _ := strings.Cut(before, ",")
	n, _ := strconv.Atoi(numStr)
	return n
}
