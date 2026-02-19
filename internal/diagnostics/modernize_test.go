package diagnostics

import (
	"strings"
	"testing"
)

func TestParseDiffEmpty(t *testing.T) {
	diags := parseDiff("")
	if len(diags) != 0 {
		t.Fatalf("expected 0, got %d", len(diags))
	}
}

func TestParseDiffBasic(t *testing.T) {
	diff := `diff main.go main.go
--- a/main.go
+++ b/main.go
@@ -3,4 +3,4 @@
 func f() {
-	interface{}
+	any
 }
`
	diags := parseDiff(diff)
	if len(diags) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(diags), diags)
	}
	if diags[0].Category != CategoryModernize {
		t.Errorf("category = %q", diags[0].Category)
	}
	if !strings.Contains(diags[0].Message, "interface{}") {
		t.Errorf("message = %q", diags[0].Message)
	}
	if diags[0].File != "main.go" {
		t.Errorf("file = %q", diags[0].File)
	}
}

func TestParseHunkStart(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"@@ -10,5 +10,5 @@", 10},
		{"@@ -1 +1 @@", 1},
		{"@@ -42,10 +42,10 @@ func foo() {", 42},
	}
	for _, tt := range tests {
		got := parseHunkStart(tt.line)
		if got != tt.want {
			t.Errorf("parseHunkStart(%q) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestBuildFixArgs(t *testing.T) {
	// Default: no fixers, no aggressive, diff mode.
	args := buildFixArgs(nil, false, true)
	last := args[len(args)-1]
	if last != "./..." {
		t.Errorf("last arg = %q, want ./...", last)
	}

	// Selective fixers.
	args = buildFixArgs([]string{"waitgroup", "rangeint"}, false, false)
	var hasDisableAll, hasWaitgroup bool
	for _, a := range args {
		if a == "-all=false" {
			hasDisableAll = true
		}
		if a == "-waitgroup=true" {
			hasWaitgroup = true
		}
	}
	if !hasDisableAll {
		t.Error("expected -all=false for selective fixers")
	}
	if !hasWaitgroup {
		t.Error("expected -waitgroup=true")
	}
}
