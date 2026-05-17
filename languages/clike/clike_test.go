package clike_test

import (
	"iter"
	"testing"

	_ "github.com/iszlai/chamele-go/languages/clike"
	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func functions(src string) []*chamele.FunctionInfo {
	r := clike.NewCLikeReader()
	a := chamele.NewFileAnalyzer()
	fi := a.AnalyzeSourceCode("test.cpp", []byte(src), r)
	return fi.Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

// Verify the interface is satisfied (compile-time check)
var _ interface {
	Tokenize([]byte) iter.Seq[string]
} = clike.NewCLikeReader()

func TestUnit_CLike_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0 functions, got %d", len(got))
	}
}

func TestUnit_CLike_NoFunction(t *testing.T) {
	if got := functions("#include <stdio.h>\n"); len(got) != 0 {
		t.Errorf("expected 0 functions, got %d: %v", len(got), names(got))
	}
}

func TestUnit_CLike_OneFunction(t *testing.T) {
	got := functions("int fun(){}")
	if len(got) != 1 {
		t.Fatalf("expected 1 function, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "fun" {
		t.Errorf("name = %q, want %q", got[0].Name, "fun")
	}
}

func TestUnit_CLike_TwoFunctions(t *testing.T) {
	got := functions("int fun(){}\nint fun1(){}\n")
	if len(got) != 2 {
		t.Fatalf("expected 2 functions, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "fun" || got[1].Name != "fun1" {
		t.Errorf("names = %v", names(got))
	}
	if got[0].StartLine != 1 || got[0].EndLine != 1 {
		t.Errorf("fun lines: %d-%d, want 1-1", got[0].StartLine, got[0].EndLine)
	}
	if got[1].StartLine != 2 || got[1].EndLine != 2 {
		t.Errorf("fun1 lines: %d-%d, want 2-2", got[1].StartLine, got[1].EndLine)
	}
}

func TestUnit_CLike_TwoSimplest(t *testing.T) {
	got := functions("f(){}g(){}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "f" || got[1].Name != "g" {
		t.Errorf("names = %v", names(got))
	}
}

func TestUnit_CLike_FunctionWithContent(t *testing.T) {
	got := functions("int fun(xx oo){int a; a= call(p1,p2);}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "fun" {
		t.Errorf("name = %q", got[0].Name)
	}
	if got[0].LongName != "fun( xx oo)" {
		t.Errorf("long_name = %q, want %q", got[0].LongName, "fun( xx oo)")
	}
}

func TestUnit_CLike_OldStyleC(t *testing.T) {
	got := functions("int fun(param) int praram; {}")
	if len(got) != 1 {
		t.Errorf("expected 1, got %d: %v", len(got), names(got))
	}
}

func TestUnit_CLike_FunctionDecWithThrow(t *testing.T) {
	got := functions("int fun() throw();void foo(){}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" {
		t.Errorf("name = %q, want foo", got[0].Name)
	}
}

func TestUnit_CLike_FunctionDecWithNoexcept(t *testing.T) {
	got := functions("int fun() noexcept(true);void foo(){}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
}

func TestUnit_CLike_FunctionDeclarationNotCounted(t *testing.T) {
	got := functions("int fun();class A{};")
	if len(got) != 0 {
		t.Errorf("expected 0, got %d: %v", len(got), names(got))
	}
}

func TestUnit_CLike_VoidParamIsZero(t *testing.T) {
	got := functions("int fun(void){}")
	if len(got) != 1 {
		t.Fatalf("expected 1 function")
	}
	if got[0].ParameterCount() != 0 {
		t.Errorf("param count = %d, want 0", got[0].ParameterCount())
	}
}

func TestUnit_CLike_OneParam(t *testing.T) {
	got := functions("int fun(aa bb){}")
	if got[0].ParameterCount() != 1 {
		t.Errorf("param count = %d, want 1", got[0].ParameterCount())
	}
}

func TestUnit_CLike_TwoParams(t *testing.T) {
	got := functions("int fun(aa * bb, cc dd){}")
	if got[0].ParameterCount() != 2 {
		t.Errorf("param count = %d, want 2", got[0].ParameterCount())
	}
}

func TestUnit_CLike_NamespacedFunction(t *testing.T) {
	got := functions("int abc::fun(){}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "abc::fun" {
		t.Errorf("name = %q, want abc::fun", got[0].Name)
	}
}

func TestUnit_CLike_ConstMethod(t *testing.T) {
	got := functions("int abc::fun()const{}")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "abc::fun" {
		t.Errorf("name = %q", got[0].Name)
	}
	if got[0].LongName != "abc::fun() const" {
		t.Errorf("long_name = %q", got[0].LongName)
	}
}

func TestUnit_CLike_Destructor(t *testing.T) {
	got := functions("class c {~c(){}}; int d(){}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "c::~c" {
		t.Errorf("names[0] = %q, want c::~c", got[0].Name)
	}
	if got[1].Name != "d" {
		t.Errorf("names[1] = %q, want d", got[1].Name)
	}
}

func TestUnit_CLike_ClassInheritance(t *testing.T) {
	got := functions("class c final:public b {int f(){}};")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "c::f" {
		t.Errorf("name = %q, want c::f", got[0].Name)
	}
}

func TestUnit_CLike_NestedClass(t *testing.T) {
	got := functions("class c {class d {int f(){}};};")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "c::d::f" {
		t.Errorf("name = %q, want c::d::f", got[0].Name)
	}
}

func TestUnit_CLike_TemplateClassWithFunction(t *testing.T) {
	got := functions("template<typename T> class c {void f(T t) {}};")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "c::f" {
		t.Errorf("name = %q, want c::f", got[0].Name)
	}
}

func TestUnit_CLike_NamespaceTemplateClass(t *testing.T) {
	got := functions("namespace ns { template<class T> class c {void f(T t) {}}; }")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "ns::c::f" {
		t.Errorf("name = %q, want ns::c::f", got[0].Name)
	}
}

func TestUnit_CLike_CCN_If(t *testing.T) {
	got := functions("int fun(){if(a){xx;}}")
	if len(got) != 1 {
		t.Fatalf("expected 1 function")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_CLike_CCN_LogicalAnd(t *testing.T) {
	got := functions("int fun(){return a && b;}")
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_CLike_CCN_RValueRef_NotCondition(t *testing.T) {
	// && in a declaration context is an r-value ref, not logical-and → CCN stays 1
	got := functions("int fun(){int && a = b;}")
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1 (&&  is r-value ref, not condition)", got[0].CyclomaticComplexity)
	}
}

func TestUnit_CLike_Macro_If_AddsCCN(t *testing.T) {
	got := functions("int fun(){\n#if a\n}")
	// #if adds 1 to CCN
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
