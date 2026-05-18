package ns

import (
	"slices"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func TestMaxNestedStructures(t *testing.T) {
	src := []byte(`
int f(int x) {
    if (x > 0) {
        if (x > 10) {
            for (int i = 0; i < x; i++) {
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
	v, _ := fi.Functions[idx].Ext[Key].(int)
	if v < 3 {
		t.Errorf("expected max_nested_structures >= 3, got %d", v)
	}
}
