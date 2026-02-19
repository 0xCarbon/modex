package diagnostics

import (
	"strings"
	"testing"
)

func TestOutdatedParseJSON(t *testing.T) {
	input := `{"Path":"github.com/foo/bar","Version":"v1.0.0","Update":{"Path":"github.com/foo/bar","Version":"v1.1.0"}}
{"Path":"github.com/baz/qux","Version":"v2.0.0"}`

	diags := parseOutdatedJSON(strings.TrimSpace(input))
	if len(diags) != 1 {
		t.Fatalf("expected 1 outdated dep, got %d", len(diags))
	}
	if !strings.Contains(diags[0].Message, "v1.0.0") {
		t.Errorf("message = %q", diags[0].Message)
	}
	if !strings.Contains(diags[0].Message, "v1.1.0") {
		t.Errorf("message = %q", diags[0].Message)
	}
	if diags[0].Category != CategoryOutdated {
		t.Errorf("category = %q", diags[0].Category)
	}
}

func TestOutdatedParseJSONNoUpdates(t *testing.T) {
	input := `{"Path":"github.com/baz/qux","Version":"v2.0.0"}`
	diags := parseOutdatedJSON(input)
	if len(diags) != 0 {
		t.Fatalf("expected 0, got %d", len(diags))
	}
}
