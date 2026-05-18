// Package nd implements the nesting-depth (ND) extension.
//
// It maintains a per-function maximum nesting depth, where a nesting level
// is opened by a control-flow keyword (if/for/while/switch/case/try/catch,
// the ternary "?", and the logical "&&"/"||" inside conditions). A level
// is closed on the matching "}" or ";". "else if" does not add a new
// level, matching upstream lizard's lizardnd.py behaviour.
//
// This is a deliberate simplification of the patched-context approach used
// by upstream Python lizard: chamele-go tracks the per-function counters on
// FunctionInfo.Ext directly rather than monkey-patching FileInfoBuilder.
package nd

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

const (
	keyDepth    = "nd_depth"
	keyPrevElse = "nd_prev_else"
	keyInCond   = "nd_in_cond"
	keyCondDep  = "nd_cond_depth"
	keyLogAdded = "nd_log_added"
)

var loopKeywords = map[string]struct{}{
	"if": {}, "for": {}, "foreach": {}, "while": {},
	"&&": {}, "||": {}, "?": {},
	"catch": {}, "case": {}, "try": {},
}

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

// New returns the ND extension instance.
func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string       { return "nd" }
func (e *ext) OrderingIndex() int { return 1000 }

func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{{
		Header: "  ND  ",
		Value:  func(f *chamele.FunctionInfo) any { return f.MaxNestingDepth },
	}}
}

func getBool(m map[string]any, k string) bool {
	if v, ok := m[k]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (e *ext) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			ext := fn.Ext

			switch tok {
			case "else":
				ext[keyPrevElse] = true
			case "{", ";":
				ext[keyPrevElse] = false
			case "(":
				ext[keyInCond] = true
				fn.Inc(keyCondDep, 1)
				ext[keyLogAdded] = false
			case ")":
				cd := fn.GetInt(keyCondDep) - 1
				if cd < 0 {
					cd = 0
				}
				ext[keyCondDep] = cd
				if cd == 0 {
					ext[keyInCond] = false
					ext[keyLogAdded] = false
				}
			}

			isLoop := false
			if _, ok := loopKeywords[tok]; ok {
				if tok == "if" && getBool(ext, keyPrevElse) {
					ext[keyPrevElse] = false
				} else if tok == "&&" || tok == "||" {
					if getBool(ext, keyInCond) {
						if !getBool(ext, keyLogAdded) {
							isLoop = true
							ext[keyLogAdded] = true
						}
					} else {
						isLoop = true
					}
				} else {
					isLoop = true
				}
			}

			if isLoop {
				d := fn.GetInt(keyDepth) + 1
				ext[keyDepth] = d
				if d > fn.MaxNestingDepth {
					fn.MaxNestingDepth = d
				}
			}

			if tok == "}" {
				d := fn.GetInt(keyDepth) - 1
				if d < 0 {
					d = 0
				}
				ext[keyDepth] = d
			}

			if !yield(tok) {
				return
			}
		}
	}
}
