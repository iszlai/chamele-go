package ttcn_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/ttcn"
)

func functions(src string) []*chamele.FunctionInfo {
	r := ttcn.NewTTCNReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.ttcn", []byte(src), r).Functions
}

func TestUnit_TTCN_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_TTCN_Function(t *testing.T) {
	got := functions("function foo(in integer a) { }")
	if len(got) != 1 || got[0].Name != "foo" {
		names := make([]string, len(got))
		for i, f := range got {
			names[i] = f.Name
		}
		t.Errorf("got %v", names)
	}
}

func TestUnit_TTCN_Testcase(t *testing.T) {
	got := functions("testcase tc_foo() runs on C { }")
	if len(got) != 1 || got[0].Name != "tc_foo" {
		t.Errorf("expected tc_foo, got %d functions", len(got))
	}
}

func TestUnit_TTCN_TwoFunctions(t *testing.T) {
	got := functions("function f() {} function g() {}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}
