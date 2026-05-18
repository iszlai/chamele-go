package ruby_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/ruby"
)

func rubyFunctions(src string) []*chamele.FunctionInfo {
	r := ruby.NewRubyReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.rb", []byte(src), r).Functions
}

func TestUnit_Ruby_Empty(t *testing.T) {
	if got := rubyFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Ruby_NoFunction(t *testing.T) {
	if got := rubyFunctions(` p "1" `); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Ruby_OneFunction(t *testing.T) {
	got := rubyFunctions(`
def f
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "f" {
		t.Errorf("name = %q, want f", got[0].Name)
	}
	if got[0].ParameterCount() != 0 {
		t.Errorf("params = %d, want 0", got[0].ParameterCount())
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Ruby_TwoFunctions(t *testing.T) {
	got := rubyFunctions(`
def f
end
def g
end
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[1].Name != "g" {
		t.Errorf("got[1].Name = %q, want g", got[1].Name)
	}
}

func TestUnit_Ruby_ClassMethod(t *testing.T) {
	got := rubyFunctions(`
def a.b
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "a.b" {
		t.Errorf("name = %q, want a.b", got[0].Name)
	}
	if got[0].ParameterCount() != 0 {
		t.Errorf("params = %d, want 0", got[0].ParameterCount())
	}
}

func TestUnit_Ruby_EmptyParameters(t *testing.T) {
	got := rubyFunctions(`
def a()
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].ParameterCount() != 0 {
		t.Errorf("params = %d, want 0", got[0].ParameterCount())
	}
}

func TestUnit_Ruby_CCN_IfElsif(t *testing.T) {
	got := rubyFunctions(`
def f
    if a
    elsif b
    end
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Ruby_CCN_BasicIf(t *testing.T) {
	got := rubyFunctions(`
def f
    if a
    end
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Ruby_BeginEnd(t *testing.T) {
	got := rubyFunctions(`
def f
    begin
        something
    end
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].NLOC != 5 {
		t.Errorf("NLOC = %d, want 5", got[0].NLOC)
	}
}

func TestUnit_Ruby_ClassInsideMethod(t *testing.T) {
	got := rubyFunctions(`
def f
    class a
    end
    module a
    end
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].NLOC != 6 {
		t.Errorf("NLOC = %d, want 6", got[0].NLOC)
	}
}
