package golang_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/golang"
)

func goFunctions(src string) []*chamele.FunctionInfo {
	r := golang.NewGoReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.go", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Go_Empty(t *testing.T) {
	if got := goFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %v", names(got))
	}
}

func TestUnit_Go_OneFunction(t *testing.T) {
	got := goFunctions("func fun(){}")
	if len(got) != 1 || got[0].Name != "fun" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Go_TwoFunctions(t *testing.T) {
	got := goFunctions("func f(){}func g(){}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "f" || got[1].Name != "g" {
		t.Errorf("names = %v", names(got))
	}
}

func TestUnit_Go_MethodReceiver(t *testing.T) {
	got := goFunctions("func (r Recv) fun(){}")
	if len(got) != 1 || got[0].Name != "fun" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Go_Parameters(t *testing.T) {
	got := goFunctions("func fun(a, b int){}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].ParameterCount() != 2 {
		t.Errorf("param count = %d, want 2", got[0].ParameterCount())
	}
}

func TestUnit_Go_TypeDefNotCounted(t *testing.T) {
	got := goFunctions("type Foo struct { Bar int }")
	if len(got) != 0 {
		t.Errorf("expected 0, got %v", names(got))
	}
}

func TestUnit_Go_InterfaceNotCounted(t *testing.T) {
	got := goFunctions("type I interface { Method() int }")
	if len(got) != 0 {
		t.Errorf("expected 0, got %v", names(got))
	}
}

func TestUnit_Go_CCN_If(t *testing.T) {
	got := goFunctions("func f(){if a {}}")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Go_CCN_For(t *testing.T) {
	got := goFunctions("func f(){for a {}}")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Go_CCN_Switch(t *testing.T) {
	got := goFunctions("func f(){switch a { case 1: }}")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Go_CCN_And(t *testing.T) {
	got := goFunctions("func f(){return a && b}")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Go_Closure(t *testing.T) {
	got := goFunctions("func outer() { g := func() {} }")
	if len(got) != 2 {
		t.Fatalf("expected 2 (outer + closure), got %d: %v", len(got), names(got))
	}
}

func TestUnit_Go_Generics(t *testing.T) {
	got := goFunctions("func Map[T, U any](s []T, f func(T) U) []U { return nil }")
	if len(got) != 1 || got[0].Name != "Map" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Go_NLOC(t *testing.T) {
	src := "func fun() {\n  a = 1\n  b = 2\n}"
	got := goFunctions(src)
	if len(got) != 1 || got[0].NLOC < 3 {
		t.Errorf("NLOC = %d, want >= 3", got[0].NLOC)
	}
}
