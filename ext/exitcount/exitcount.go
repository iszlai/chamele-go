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
			if f.Ext != nil {
				if v, ok := f.Ext[key]; ok {
					return v
				}
			}
			return 0
		},
	}}
}

func (e *exitCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	firstReturn := false
	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			if _, ok := fn.Ext[key]; !ok {
				fn.Ext[key] = 1
				firstReturn = true
			}
			if tok == "return" || tok == "exit" {
				if firstReturn {
					firstReturn = false
				} else {
					fn.Ext[key] = fn.Ext[key].(int) + 1
				}
			}
			if !yield(tok) {
				return
			}
		}
	}
}
