package gdscript_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/gdscript"
)

func gdFunctions(src string) []*chamele.FunctionInfo {
	r := gdscript.NewGDScriptReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.gd", []byte(src), r).Functions
}

func TestUnit_GDScript_Empty(t *testing.T) {
	if got := gdFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_GDScript_TopLevelFunction(t *testing.T) {
	got := gdFunctions("func a():\n    pass")
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "a" {
		t.Errorf("name = %q, want %q", got[0].Name, "a")
	}
}

func TestUnit_GDScript_ElIfKeyword(t *testing.T) {
	src := `
func test_if_else(x, y):
    if x > 0:
        print("positive")
    else:
        print("not positive")

    if y > 10:
        return "high"
    elif y > 5:
        return "medium"
    else:
        return "low"
`
	got := gdFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// base(1) + if(1) + if(1) + elif(1) = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}
