// Package gotocount counts goto statements per function.
package gotocount

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

const key = "goto_count"

type gotoCountExt struct{}

func New() chamele.Extension { return &gotoCountExt{} }

func (e *gotoCountExt) Name() string       { return key }
func (e *gotoCountExt) OrderingIndex() int { return 1000 }

func (e *gotoCountExt) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{{
		Header: "goto's",
		Value:  func(f *chamele.FunctionInfo) any { return f.GetInt(key) },
	}}
}

func (e *gotoCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "goto" {
				ctx.CurrentFunction.Inc(key, 1)
			}
			if !yield(tok) {
				return
			}
		}
	}
}
