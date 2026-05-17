package solidity_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/solidity"
)

func functions(src string) []*chamele.FunctionInfo {
	r := solidity.NewSolidityReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.sol", []byte(src), r).Functions
}

func TestUnit_Solidity_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Solidity_SimpleFunction(t *testing.T) {
	got := functions("contract C { function foo() public { } }")
	if len(got) != 1 || got[0].Name != "foo" {
		t.Errorf("got %v", func() []string {
			n := make([]string, len(got))
			for i, f := range got { n[i] = f.Name }
			return n
		}())
	}
}

func TestUnit_Solidity_TwoFunctions(t *testing.T) {
	got := functions("function foo() {} function bar() {}")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}

func TestUnit_Solidity_CCN_If(t *testing.T) {
	got := functions("function f() { if (a) {} }")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
