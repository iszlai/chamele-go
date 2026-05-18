package mccabe

import (
	"slices"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func TestStrictMcCabeFallthroughCase(t *testing.T) {
	src := []byte(`
int f(int x) {
    switch (x) {
        case 1:
        case 2:
        case 3:
            return x;
        default:
            return 0;
    }
}
`)
	r := clike.NewCLikeReader()
	a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{New()})
	fi := a.AnalyzeSourceCode("t.c", src, r)

	idx := slices.IndexFunc(fi.Functions, func(f *chamele.FunctionInfo) bool { return f.Name == "f" })
	if idx < 0 {
		t.Fatal("function f not found")
	}
	// With strict McCabe, consecutive case 1/2/3 fold into one branch.
	// Default conditions: 1 (base) + 1 (switch baseline) + 1 (single case branch).
	// Without mccabe extension lizard would give 4 (3 cases). We assert <4.
	if fi.Functions[idx].CyclomaticComplexity > 3 {
		t.Errorf("strict McCabe expected CCN<=3, got %d", fi.Functions[idx].CyclomaticComplexity)
	}
}
