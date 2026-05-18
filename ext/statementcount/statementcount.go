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
		Value:  func(f *chamele.FunctionInfo) any { return f.GetInt(key) },
	}}
}

func (e *statementCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		blockCount := 0
		for tok := range tokens {
			fn := ctx.CurrentFunction
			switch tok {
			case ";", "if", "for", "while":
				fn.Inc(key, 1)
			case "{":
				if blockCount > 0 {
					fn.Inc(key, 1)
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
