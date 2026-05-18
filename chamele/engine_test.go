package chamele

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestEngine_WithExtensions_OverridesGlobal verifies that the
// WithExtensions option suppresses the global registry — useful for library
// callers running a focused subset.
func TestEngine_WithExtensions_OverridesGlobal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.c", "int f() { return 1; }\n")

	// stateful from extension_test.go: counts every token it sees.
	count := &stateful{}
	e := New(WithExtensions(count))
	files, err := e.AnalyzePaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if len(e.Extensions()) != 1 {
		t.Errorf("expected one extension instance, got %d", len(e.Extensions()))
	}
	if e.Extensions()[0] != count {
		t.Errorf("WithExtensions did not override the global registry")
	}
	if count.count == 0 {
		t.Error("expected the override extension to have processed tokens")
	}
}

// TestEngine_TwoRuns_IndependentMetrics verifies the Phase D + Phase E
// guarantee: instantiating two Engines (or calling AnalyzeWithExtensions
// twice) produces independent per-run extension state.
func TestEngine_TwoRuns_IndependentMetrics(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.c", "int f() { return 1; }\nint g() { return 2; }\n")

	a := &stateful{}
	b := &stateful{}

	_, err := New(WithExtensions(a)).AnalyzePaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(WithExtensions(b)).AnalyzePaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	if a.count == 0 || b.count == 0 {
		t.Fatalf("expected both runs to count tokens, got a=%d b=%d", a.count, b.count)
	}
	if a.count != b.count {
		t.Errorf("two runs over identical input gave different counts: a=%d b=%d", a.count, b.count)
	}
}
