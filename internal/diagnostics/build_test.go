package diagnostics

import (
	"testing"
)

func TestParseDiagLinesEmpty(t *testing.T) {
	diags := parseDiagLines("", CategoryBuild, SeverityError)
	if len(diags) != 0 {
		t.Fatalf("expected 0, got %d", len(diags))
	}
}

func TestParseDiagLinesCompileError(t *testing.T) {
	input := `./main.go:5:10: undefined: foo
./main.go:8:1: cannot use x (type int) as type string`
	diags := parseDiagLines(input, CategoryBuild, SeverityError)
	if len(diags) != 2 {
		t.Fatalf("expected 2, got %d", len(diags))
	}
	if diags[0].File != "./main.go" {
		t.Errorf("file = %q", diags[0].File)
	}
	if diags[0].Line != 5 {
		t.Errorf("line = %d", diags[0].Line)
	}
	if diags[0].Category != CategoryBuild {
		t.Errorf("category = %q", diags[0].Category)
	}
	if diags[0].Severity != SeverityError {
		t.Errorf("severity = %q", diags[0].Severity)
	}
}

func TestParseDiagLinesNoColumn(t *testing.T) {
	input := `./pkg/foo.go:12: syntax error: unexpected }`
	diags := parseDiagLines(input, CategoryBuild, SeverityError)
	if len(diags) != 1 {
		t.Fatalf("expected 1, got %d", len(diags))
	}
	if diags[0].Line != 12 {
		t.Errorf("line = %d", diags[0].Line)
	}
}

func TestParseDiagLinesSkipsNonMatchLines(t *testing.T) {
	input := `# github.com/0xCarbon/modex/cmd/modex
build failed
FAIL    github.com/0xCarbon/modex/cmd/modex [build failed]`
	diags := parseDiagLines(input, CategoryBuild, SeverityError)
	if len(diags) != 0 {
		t.Fatalf("expected 0, got %d: %v", len(diags), diags)
	}
}
