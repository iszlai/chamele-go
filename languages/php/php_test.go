package php_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/php"
)

func functions(src string) []*chamele.FunctionInfo {
	r := php.NewPHPReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.php", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_PHP_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_PHP_SimpleFunction(t *testing.T) {
	got := functions("<?php function foo() {} ?>")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "foo" {
		t.Errorf("name = %q, want foo", got[0].Name)
	}
}

func TestUnit_PHP_TwoFunctions(t *testing.T) {
	got := functions("<?php function foo() {} function bar() {} ?>")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_PHP_NoPhpBlock_Empty(t *testing.T) {
	// Code outside <?php ... ?> is ignored
	got := functions("function foo() {}")
	if len(got) != 0 {
		t.Errorf("expected 0 (no PHP block), got %d: %v", len(got), names(got))
	}
}

func TestUnit_PHP_CCN_If(t *testing.T) {
	got := functions("<?php function foo() { if ($x) {} } ?>")
	if len(got) != 1 {
		t.Fatalf("expected 1")
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
