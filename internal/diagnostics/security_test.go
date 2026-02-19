package diagnostics

import "testing"

func TestParseGovulncheckJSONFinding(t *testing.T) {
	input := `{"type":"Finding","data":{"osv":"GO-2024-0001","trace":[{"position":"pkg/foo.go:12:3"}]}}`

	diags := parseGovulncheckJSON(input)
	if len(diags) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(diags))
	}
	if diags[0].Category != CategorySecurity {
		t.Fatalf("category = %q", diags[0].Category)
	}
	if diags[0].Message != "GO-2024-0001" {
		t.Fatalf("message = %q", diags[0].Message)
	}
	if diags[0].File != "pkg/foo.go:12:3" {
		t.Fatalf("file = %q", diags[0].File)
	}
}

func TestParseGovulncheckJSONStopsOnMalformedRecord(t *testing.T) {
	input := `{"type":"Finding","data":{"osv":"GO-2024-0001","trace":[{"position":"pkg/foo.go:12:3"}]}}
not-json
{"type":"Finding","data":{"osv":"GO-2024-0002","trace":[{"position":"pkg/bar.go:3:1"}]}}`

	diags := parseGovulncheckJSON(input)
	if len(diags) != 1 {
		t.Fatalf("expected 1 finding before malformed record, got %d", len(diags))
	}
	if diags[0].Message != "GO-2024-0001" {
		t.Fatalf("unexpected first finding: %#v", diags[0])
	}
}
