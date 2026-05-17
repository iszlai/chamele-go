package csharp_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/csharp"
)

func functions(src string) []*chamele.FunctionInfo {
	r := csharp.NewCSharpReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.cs", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_CSharp_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0 functions, got %d", len(got))
	}
}

func TestUnit_CSharp_SimpleMethod(t *testing.T) {
	got := functions("void fun() {}")
	if len(got) != 1 {
		t.Fatalf("expected 1 function, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "fun" {
		t.Errorf("name = %q, want fun", got[0].Name)
	}
}

func TestUnit_CSharp_ClassMethod(t *testing.T) {
	got := functions("class A { void fun() {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1 function, got %d: %v", len(got), names(got))
	}
}

func TestUnit_CSharp_TwoMethods(t *testing.T) {
	got := functions("void foo(){} void bar(){}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" || got[1].Name != "bar" {
		t.Errorf("names = %v", names(got))
	}
}

func TestUnit_CSharp_CCN_If(t *testing.T) {
	got := functions("void fun() { if (a) {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1 function")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_CSharp_NullCoalescing_NotCondition(t *testing.T) {
	// ?? is a null-coalescing operator; it contributes to CCN
	got := functions("void fun() { var x = a ?? b; }")
	if len(got) != 1 {
		t.Fatalf("expected 1 function")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2 (?? counts)", got[0].CyclomaticComplexity)
	}
}
