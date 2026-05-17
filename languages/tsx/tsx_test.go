package tsx_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/tsx"
)

func functions(src string) []*chamele.FunctionInfo {
	r := tsx.NewTSXReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.tsx", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_TSX_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_TSX_SimpleFunction(t *testing.T) {
	got := functions("function Foo() {}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "Foo" {
		t.Errorf("name = %q, want Foo", got[0].Name)
	}
}

func TestUnit_TSX_AssignedFunction(t *testing.T) {
	got := functions("const Bar = function() { }")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "Bar" {
		t.Errorf("name = %q, want Bar", got[0].Name)
	}
}

func TestUnit_TSX_TwoFunctions(t *testing.T) {
	got := functions("function Foo() {} function Bar() {}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_TSX_CCN_If(t *testing.T) {
	got := functions("function foo() { if (x) {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
