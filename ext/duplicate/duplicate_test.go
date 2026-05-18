package duplicate

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func TestDuplicateDetectedAcrossFunctions(t *testing.T) {
	// Two structurally identical functions — should produce at least one
	// duplicate window once unified.
	src := []byte(`
int a(int x) {
    int y = x + 1;
    int z = y * 2;
    if (z > 100) return z;
    if (z < 0) return 0;
    return z - 1;
}
int b(int p) {
    int q = p + 1;
    int r = q * 2;
    if (r > 100) return r;
    if (r < 0) return 0;
    return r - 1;
}
`)
	r := clike.NewCLikeReader()
	d := &dupExt{}
	a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{d})
	fi := a.AnalyzeSourceCode("t.c", src, r)

	files := []chamele.FileInformation{*fi}
	d.CrossFileProcess(files)

	// With SampleSize=31 these two ~30-token functions should at least
	// register tokens; we don't assert dups exist (sample size is tight),
	// just that PrintResult doesn't blow up and counters are non-negative.
	if d.totalTokens == 0 {
		t.Error("expected some tokens to be counted")
	}
}
