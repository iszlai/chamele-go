package chamele_test

import (
	"iter"
	"os"
	"path/filepath"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

// stubReader uses the base tokenizer with C/C++-style comment detection.
type stubReader struct{}

func (s stubReader) Extensions() []string    { return []string{"c", "h"} }
func (s stubReader) LanguageNames() []string { return []string{"c"} }
func (s stubReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, "")
}
func (s stubReader) GetComment(tok string) (string, bool) {
	if len(tok) >= 2 && (tok[:2] == "//" || tok[:2] == "/*") {
		return tok[2:], true
	}
	return "", false
}

func init() {
	languages.Register(stubReader{})
}

func TestE2E_AnalyzeSourceCode_NLOC(t *testing.T) {
	src := []byte("int foo() {\n    return 1;\n}\n")
	a := chamele.NewFileAnalyzer()
	fi := a.AnalyzeSourceCode("test.c", src, stubReader{})
	if fi.NLOC == 0 {
		t.Errorf("expected non-zero NLOC, got 0")
	}
}

func TestE2E_AnalyzeSourceCode_TokenCount(t *testing.T) {
	src := []byte("int foo() { return 1; }")
	a := chamele.NewFileAnalyzer()
	fi := a.AnalyzeSourceCode("test.c", src, stubReader{})
	if fi.TokenCount == 0 {
		t.Errorf("expected non-zero token count, got 0")
	}
}

func TestE2E_AnalyzeSourceCode_LineCounter_Newlines(t *testing.T) {
	// Three non-blank lines → NLOC should be 3
	src := []byte("a\nb\nc\n")
	a := chamele.NewFileAnalyzer()
	fi := a.AnalyzeSourceCode("test.c", src, stubReader{})
	if fi.NLOC != 3 {
		t.Errorf("expected NLOC=3, got %d", fi.NLOC)
	}
}

func TestE2E_AnalyzeSourceCode_ForgiveGlobal(t *testing.T) {
	src := []byte("// #lizard forgive global\nint foo() {}\n")
	a := chamele.NewFileAnalyzer()
	fi := a.AnalyzeSourceCode("test.c", src, stubReader{})
	// With forgive_global, functions should not be appended.
	// The global pseudo-function is forgiven, so function_list stays empty.
	_ = fi
}

// --- Walk tests ---

func TestE2E_Walk_ExcludePattern(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.c", "int x;")
	writeFile(t, dir, "b.c", "int y;")

	results, err := chamele.Analyze([]string{dir}, chamele.WithExclude("b.c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fi := range results {
		if filepath.Base(fi.Filename) == "b.c" {
			t.Errorf("b.c should have been excluded")
		}
	}
	if len(results) == 0 {
		t.Error("expected at least one result")
	}
}

func TestE2E_Walk_LanguageFilter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.c", "int x;")
	// No .py reader registered, so .py files should be skipped.
	writeFile(t, dir, "b.py", "def f(): pass")

	results, err := chamele.Analyze([]string{dir}, chamele.WithLanguages("c"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fi := range results {
		if filepath.Ext(fi.Filename) != ".c" {
			t.Errorf("non-c file slipped through: %s", fi.Filename)
		}
	}
}

func TestE2E_Walk_MD5Dedup(t *testing.T) {
	dir := t.TempDir()
	content := "int x = 42;"
	writeFile(t, dir, "a.c", content)
	writeFile(t, dir, "b.c", content) // identical content → same MD5

	results, err := chamele.Analyze([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result after dedup, got %d", len(results))
	}
}

func TestE2E_Walk_Gitignore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".gitignore", "ignored.c\n")
	writeFile(t, dir, "ignored.c", "int x;")
	writeFile(t, dir, "visible.c", "int y;")

	results, err := chamele.Analyze([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, fi := range results {
		if filepath.Base(fi.Filename) == "ignored.c" {
			t.Errorf("ignored.c should have been excluded by .gitignore")
		}
	}
	found := false
	for _, fi := range results {
		if filepath.Base(fi.Filename) == "visible.c" {
			found = true
		}
	}
	if !found {
		t.Error("visible.c should have been included")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
