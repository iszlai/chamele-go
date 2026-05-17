package perl_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/perl"
)

func functions(src string) []*chamele.FunctionInfo {
	r := perl.NewPerlReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.pl", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Perl_Empty(t *testing.T) {
	if got := functions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Perl_SimpleFunction(t *testing.T) {
	got := functions("sub foo { }")
	if len(got) != 1 || got[0].Name != "foo" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Perl_TwoFunctions(t *testing.T) {
	got := functions("sub foo { }\nsub bar { }")
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), names(got))
	}
}

func TestUnit_Perl_AnonymousSub(t *testing.T) {
	got := functions("my $f = sub { }")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
}

func TestUnit_Perl_ForwardDeclaration(t *testing.T) {
	// Forward declarations (sub foo;) should NOT create functions.
	got := functions("sub foo; sub bar { }")
	if len(got) != 1 || got[0].Name != "bar" {
		t.Errorf("expected [bar], got %v", names(got))
	}
}

func TestUnit_Perl_CCN_If(t *testing.T) {
	got := functions("sub f { if ($a) { } }")
	if len(got) != 1 || got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
