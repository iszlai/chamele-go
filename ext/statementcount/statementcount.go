// Package statementcount counts statements (semicolons) per function.
package statementcount

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

const key = "statement_count"

type statementCountExt struct{}

func New() chamele.Extension { return &statementCountExt{} }

func (e *statementCountExt) Name() string       { return key }
func (e *statementCountExt) OrderingIndex() int { return 1000 }

func (e *statementCountExt) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{{
		Header: "statements",
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

func (e *statementCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	blockCount := 0
	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			if _, ok := fn.Ext[key]; !ok {
				fn.Ext[key] = 0
			}
			switch tok {
			case ";", "if", "for", "while":
				fn.Ext[key] = fn.Ext[key].(int) + 1
			case "{":
				if blockCount > 0 {
					fn.Ext[key] = fn.Ext[key].(int) + 1
				}
				blockCount++
			case "}":
				if blockCount > 0 {
					blockCount--
				}
			}
			if !yield(tok) {
				return
			}
		}
	}
}
