// Package exitcount counts exit points (return/exit statements) per function.
package exitcount

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

const key = "exit_count"

type exitCountExt struct{}

func New() chamele.Extension { return &exitCountExt{} }

func (e *exitCountExt) Name() string       { return key }
func (e *exitCountExt) OrderingIndex() int { return 1000 }

func (e *exitCountExt) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{{
		Header: "exits",
		Value: func(f *chamele.FunctionInfo) any {
			if v := f.GetInt(key); v > 0 {
				return v
			}
			return 1
		},
	}}
}

func (e *exitCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "return" || tok == "exit" {
				ctx.CurrentFunction.Inc(key, 1)
			}
			if !yield(tok) {
				return
			}
		}
	}
}
