package io

import (
	"slices"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/clike"
)

func TestFanInOut(t *testing.T) {
	srcA := []byte(`
int helper(int x) { return x + 1; }
int caller(int x) { return helper(x) + helper(x); }
`)
	r := clike.NewCLikeReader()
	a := chamele.NewFileAnalyzerWithExts([]chamele.Extension{New()})
	fi := a.AnalyzeSourceCode("a.c", srcA, r)

	files := []chamele.FileInformation{*fi}
	files = New().(*ext).CrossFileProcess(files) // direct call, simulating Analyze pipeline

	idxH := slices.IndexFunc(files[0].Functions, func(fn *chamele.FunctionInfo) bool { return fn.Name == "helper" })
	idxC := slices.IndexFunc(files[0].Functions, func(fn *chamele.FunctionInfo) bool { return fn.Name == "caller" })
	if idxH < 0 || idxC < 0 {
		t.Fatal("functions not found")
	}
	if files[0].Functions[idxH].FanIn < 2 {
		t.Errorf("helper FanIn should be >= 2, got %d", files[0].Functions[idxH].FanIn)
	}
	if files[0].Functions[idxC].FanOut < 2 {
		t.Errorf("caller FanOut should be >= 2, got %d", files[0].Functions[idxC].FanOut)
	}
}
