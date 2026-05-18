package vue_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/vue"
)

func vueFunctions(src string) []*chamele.FunctionInfo {
	r := vue.NewVueReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.vue", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Vue_Empty(t *testing.T) {
	if got := vueFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Vue_SimpleJSFunction(t *testing.T) {
	src := `
<template><div>Hello</div></template>
<script>
export default {
    methods: {
        hello() {
            return "world"
        }
    }
}
</script>
`
	got := vueFunctions(src)
	if len(got) != 1 || got[0].Name != "hello" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Vue_MultipleFunctions(t *testing.T) {
	src := `
<script>
function helper1() {
    return 1;
}
export default {
    methods: {
        method1() {
            return helper1();
        },
        method2() {
            if (true) {
                return 2;
            }
            return 3;
        }
    }
}
</script>
`
	got := vueFunctions(src)
	wantNames := []string{"helper1", "method1", "method2"}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d: %v", len(got), names(got))
	}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Errorf("got[%d].Name = %q, want %q", i, got[i].Name, want)
		}
	}
	if got[2].CyclomaticComplexity != 2 {
		t.Errorf("method2 CCN = %d, want 2", got[2].CyclomaticComplexity)
	}
}
