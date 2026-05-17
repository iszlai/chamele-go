package zig_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/zig"
)

func functions(src string) []*chamele.FunctionInfo {
	r := zig.NewZigReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.zig", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Zig_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Zig_SimpleFunction(t *testing.T) {
	got := functions("fn foo() void {}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" {
		t.Errorf("name = %q, want foo", got[0].Name)
	}
}

func TestUnit_Zig_TwoFunctions(t *testing.T) {
	got := functions("fn foo() void {} fn bar() i32 {}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_Zig_FunctionWithParams(t *testing.T) {
	got := functions("fn add(a: i32, b: i32) i32 { return a + b; }")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "add" {
		t.Errorf("name = %q, want add", got[0].Name)
	}
}

func TestUnit_Zig_CCN_If(t *testing.T) {
	got := functions("fn foo() void { if (x) {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
