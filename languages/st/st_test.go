package st_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/st"
)

func stFunctions(src string) []*chamele.FunctionInfo {
	r := st.NewSTReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.st", []byte(src), r).Functions
}

func TestUnit_ST_Empty(t *testing.T) {
	if got := stFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_ST_NoFunction(t *testing.T) {
	if got := stFunctions("(* Comment1 *)\n"); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_ST_OneFunction(t *testing.T) {
	src := "(* Comment *)\nFUNCTION fun:\n// Comment\nfoo := bar;\nEND_FUNCTION\n"
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "fun" {
		t.Errorf("name = %q, want fun", got[0].Name)
	}
}

func TestUnit_ST_TwoFunctions(t *testing.T) {
	src := "FUNCTION fun1:\n// Comment\nfoo := bar;\nEND_FUNCTION\n" +
		"(* Comment *)\nFUNCTION fun2:\n// Comment\nfoo := bar;\nEND_FUNCTION\n"
	got := stFunctions(src)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "fun1" || got[1].Name != "fun2" {
		t.Errorf("names = [%q, %q], want [fun1, fun2]", got[0].Name, got[1].Name)
	}
	if got[0].StartLine != 1 || got[0].EndLine != 4 {
		t.Errorf("fun1 lines = %d-%d, want 1-4", got[0].StartLine, got[0].EndLine)
	}
	if got[1].StartLine != 6 || got[1].EndLine != 9 {
		t.Errorf("fun2 lines = %d-%d, want 6-9", got[1].StartLine, got[1].EndLine)
	}
}

func TestUnit_ST_OneAction(t *testing.T) {
	src := "(* Comment *)\nACTION ac1:\n// Comment\nfoo := bar;\nEND_ACTION\n"
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "ac1" {
		t.Errorf("name = %q, want ac1", got[0].Name)
	}
}

func TestUnit_ST_TwoActions(t *testing.T) {
	src := "ACTION ac1:\n// Comment\nfoo := bar;\nEND_ACTION\n" +
		"(* Comment *)\nACTION ac2:\n// Comment\nfoo := bar;\nEND_ACTION\n"
	got := stFunctions(src)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "ac1" || got[1].Name != "ac2" {
		t.Errorf("names = [%q, %q], want [ac1, ac2]", got[0].Name, got[1].Name)
	}
}

func TestUnit_ST_CCN_If(t *testing.T) {
	src := `
ACTION ac1:
    a := 200;
    IF b THEN
        c := 1;
    END_IF
END_ACTION
`
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_ST_CCN_Case(t *testing.T) {
	src := `
ACTION ac1:
    a := 200;
    CASE a OF
        b1:
            c := 1;
        b2:
            c := 2;
    END_CASE
END_ACTION
`
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_ST_CCN_For(t *testing.T) {
	src := `
ACTION ac1:
    a := 10;
    FOR b := 1 TO a DO
        c := c + b;
    END_FOR
END_ACTION
`
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_ST_CCN_While(t *testing.T) {
	src := `
ACTION ac1:
    a := 10;
    WHILE b < a DO
        c := c + b;
        b := b + 1;
    END_WHILE
END_ACTION
`
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_ST_CCN_Repeat(t *testing.T) {
	src := `
ACTION ac1:
    REPEAT
        c := c + b;
        b := b + 1;
        UNTIL b < a
    END_REPEAT
END_ACTION
`
	got := stFunctions(src)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_ST_PreprocessorNotFunction(t *testing.T) {
	src := `
#ifdef A
#elif (defined E)
#endif
`
	if got := stFunctions(src); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}
