package python_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/python"
)

func pyFunctions(src string) []*chamele.FunctionInfo {
	r := python.NewPythonReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.py", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Python_Empty(t *testing.T) {
	if got := pyFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %v", names(got))
	}
}

// All functions need indented bodies for Python's indentation-based detection.

func TestUnit_Python_OneFunction(t *testing.T) {
	src := "def fun():\n    pass\n"
	got := pyFunctions(src)
	if len(got) != 1 || got[0].Name != "fun" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Python_TwoFunctions(t *testing.T) {
	src := "def f():\n    pass\ndef g():\n    pass\n"
	got := pyFunctions(src)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_Python_OneParam(t *testing.T) {
	got := pyFunctions("def fun(a):\n    pass\n")
	if len(got) != 1 || got[0].ParameterCount() != 1 {
		t.Errorf("param count = %d, want 1", got[0].ParameterCount())
	}
}

func TestUnit_Python_TwoParams(t *testing.T) {
	got := pyFunctions("def fun(a, b):\n    pass\n")
	if len(got) != 1 || got[0].ParameterCount() != 2 {
		t.Errorf("param count = %d, want 2", got[0].ParameterCount())
	}
}

func TestUnit_Python_CCN_If(t *testing.T) {
	src := "def f():\n    if a:\n        pass\n"
	got := pyFunctions(src)
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Python_CCN_Elif(t *testing.T) {
	src := "def f():\n    if a:\n        pass\n    elif b:\n        pass\n"
	got := pyFunctions(src)
	if len(got) != 1 || got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Python_CCN_And(t *testing.T) {
	src := "def f():\n    return a and b\n"
	got := pyFunctions(src)
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Python_NestedFunction(t *testing.T) {
	src := "def outer():\n    def inner():\n        pass\n"
	got := pyFunctions(src)
	if len(got) < 1 {
		t.Errorf("expected at least 1 function, got %v", names(got))
	}
}

func TestUnit_Python_NLOC(t *testing.T) {
	src := "def f():\n    a = 1\n    b = 2\n    return a + b\n"
	got := pyFunctions(src)
	if len(got) != 1 || got[0].NLOC < 3 {
		t.Errorf("NLOC = %d, want >= 3", got[0].NLOC)
	}
}
