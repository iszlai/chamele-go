// Package io implements the fan-in / fan-out extension.
//
// During per-file token processing each function records the set of tokens
// it references (saved on FunctionInfo.Ext["io_tokens"]). After every file
// has been analyzed, CrossFileProcess walks all functions and:
//
//   - fan_out: for each token in fn that matches the unqualified name of
//     another function in the corpus, increment fn's fan_out.
//   - fan_in:  for each function name referenced inside fn, increment that
//     callee's fan_in.
//   - general_fan_out: count of "(" tokens preceded by a non-keyword
//     identifier (an approximation of "calls some function").
//
// Derived from upstream lizard_ext/lizardio.py.
package io

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

const TokensKey = "io_tokens"

var (
	ioStructures = map[string]struct{}{
		"if": {}, "else": {}, "elif": {}, "for": {}, "foreach": {},
		"while": {}, "do": {}, "try": {}, "catch": {}, "switch": {},
		"finally": {}, "except": {}, "with": {},
	}
	ioPunct = map[string]struct{}{
		"(": {}, ")": {}, "{": {}, "}": {},
	}
)

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

// New returns the fan-in/fan-out extension instance.
func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string       { return "io" }
func (e *ext) OrderingIndex() int { return 1000 }

func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{
		{Header: " fan_in ", Value: func(f *chamele.FunctionInfo) any { return f.FanIn }},
		{Header: " fan_out ", Value: func(f *chamele.FunctionInfo) any { return f.FanOut }},
		{Header: " general_fan_out ", Value: func(f *chamele.FunctionInfo) any { return f.GeneralFanOut }},
	}
}

// Process records every token seen while a function is current.
func (e *ext) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			lst, _ := fn.Ext[TokensKey].([]string)
			fn.Ext[TokensKey] = append(lst, tok)
			if !yield(tok) {
				return
			}
		}
	}
}

// CrossFileProcess computes fan-in, fan-out, and general-fan-out across all
// analysed files.
func (e *ext) CrossFileProcess(files []chamele.FileInformation) []chamele.FileInformation {
	all := make(map[string]*chamele.FunctionInfo)
	for i := range files {
		for _, fn := range files[i].Functions {
			all[fn.UnqualifiedName()] = fn
		}
	}

	for _, callee := range all {
		toks, _ := callee.Ext[TokensKey].([]string)
		// general_fan_out: count of "(" preceded by an identifier that
		// isn't a keyword or punctuation.
		for i := 1; i < len(toks); i++ {
			if toks[i] != "(" {
				continue
			}
			prev := toks[i-1]
			if _, ok := ioStructures[prev]; ok {
				continue
			}
			if _, ok := ioPunct[prev]; ok {
				continue
			}
			callee.GeneralFanOut++
		}
		// fan_out for callee: every other-function name appearing in callee's tokens.
		for _, tok := range toks {
			if other, ok := all[tok]; ok && other != callee {
				callee.FanOut++
				other.FanIn++
			}
		}
	}

	return files
}
