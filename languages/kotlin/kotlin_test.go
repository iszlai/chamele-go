package kotlin_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/kotlin"
)

func functions(src string) []*chamele.FunctionInfo {
	r := kotlin.NewKotlinReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.kt", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Kotlin_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Kotlin_SimpleFunction(t *testing.T) {
	got := functions("fun foo() {}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" {
		t.Errorf("name = %q, want foo", got[0].Name)
	}
}

func TestUnit_Kotlin_TwoFunctions(t *testing.T) {
	got := functions("fun foo() {} fun bar() {}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" || got[1].Name != "bar" {
		t.Errorf("names = %v", names(got))
	}
}

func TestUnit_Kotlin_FunctionWithParams(t *testing.T) {
	got := functions("fun add(a: Int, b: Int): Int { return a + b }")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "add" {
		t.Errorf("name = %q, want add", got[0].Name)
	}
}

func TestUnit_Kotlin_CCN_If(t *testing.T) {
	got := functions("fun foo() { if (x) {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
