package nd

import (
	"slices"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func TestMaxNestingDepth(t *testing.T) {
	src := []byte(`
int f(int x) {
    if (x > 0) {
        if (x > 10) {
            while (x > 0) {
                x--;
            }
        }
    }
    return x;
}
`)
	r := clike.NewCLikeReader()
	a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{New()})
	fi := a.AnalyzeSourceCode("t.c", src, r)

	idx := slices.IndexFunc(fi.Functions, func(fn *chamele.FunctionInfo) bool { return fn.Name == "f" })
	if idx < 0 {
		t.Fatal("function f not found")
	}
	got := fi.Functions[idx].MaxNestingDepth
	if got < 3 {
		t.Errorf("expected MaxNestingDepth >= 3, got %d", got)
	}
}

func TestElseIfNotNested(t *testing.T) {
	src := []byte(`
int g(int x) {
    if (x == 0) {
        return 0;
    } else if (x == 1) {
        return 1;
    } else {
        return 2;
    }
}
`)
	r := clike.NewCLikeReader()
	a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{New()})
	fi := a.AnalyzeSourceCode("t.c", src, r)

	idx := slices.IndexFunc(fi.Functions, func(fn *chamele.FunctionInfo) bool { return fn.Name == "g" })
	if idx < 0 {
		t.Fatal("function g not found")
	}
	// else-if should NOT add an extra nesting level — depth stays at 1.
	if fi.Functions[idx].MaxNestingDepth > 1 {
		t.Errorf("else-if should not increase depth; got %d", fi.Functions[idx].MaxNestingDepth)
	}
}
