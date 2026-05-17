package javascript_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/javascript"
)

func jsFunctions(src string) []*chamele.FunctionInfo {
	r := javascript.NewJSReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.js", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_JS_SimpleFunction(t *testing.T) {
	got := jsFunctions("function foo(){}")
	if len(got) != 1 || got[0].Name != "foo" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_JS_SimpleFunctionCCN(t *testing.T) {
	got := jsFunctions("function foo(){m;if(a);}")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_JS_ParameterCount(t *testing.T) {
	got := jsFunctions("function foo(a, b){}")
	if len(got) != 1 || got[0].ParameterCount() != 2 {
		t.Errorf("param count = %d, want 2", got[0].ParameterCount())
	}
}

func TestUnit_JS_AssignedFunction(t *testing.T) {
	got := jsFunctions("a = function (a, b){}")
	if len(got) != 1 || got[0].Name != "a" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_JS_AnonymousFunction(t *testing.T) {
	got := jsFunctions("abc=3; function (a, b){}")
	if len(got) != 1 || got[0].Name != "(anonymous)" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_JS_NestedFunctions(t *testing.T) {
	got := jsFunctions("function a(){function b(){}}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_JS_CCN_LogicalAnd(t *testing.T) {
	got := jsFunctions("function f(){ return a && b; }")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
