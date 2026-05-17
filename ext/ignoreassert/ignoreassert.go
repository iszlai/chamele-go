// Package ignoreassert drops CCN conditions inside assert(...) calls.
package ignoreassert

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type ignoreAssertExt struct{}

func New() chamele.Extension { return &ignoreAssertExt{} }

func (e *ignoreAssertExt) Name() string        { return "ignoreassert" }
func (e *ignoreAssertExt) OrderingIndex() int  { return 900 }
func (e *ignoreAssertExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

func (e *ignoreAssertExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		depth := 0
		inAssert := false
		for tok := range tokens {
			if tok == "assert" || tok == "static_assert" {
				inAssert = true
				depth = 0
				if !yield(tok) {
					return
				}
				continue
			}
			if inAssert {
				if tok == "(" {
					depth++
				} else if tok == ")" {
					depth--
					if depth == 0 {
						inAssert = false
					}
				} else if isCondition(tok) {
					// Cancel the +1 that ConditionCounter already added.
					ctx.AddCondition(-1)
				}
				if !yield(tok) {
					return
				}
				continue
			}
			if !yield(tok) {
				return
			}
		}
	}
}

func isCondition(tok string) bool {
	return tok == "&&" || tok == "||" || tok == "?"
}
