// Package modified implements the modified CCN extension.
// Switch/case is counted as 1 (instead of case contributing +1 each).
package modified

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type modifiedExt struct{}

func New() chamele.Extension { return &modifiedExt{} }

func (e *modifiedExt) Name() string        { return "modified" }
func (e *modifiedExt) OrderingIndex() int  { return 1000 }
func (e *modifiedExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

// Process runs after the standard ConditionCounter. For each `switch` token
// it adds +1 (switch itself isn't in the default condition set). For each
// `case` token it subtracts -1 (cancels the +1 that ConditionCounter added),
// so the whole switch/case construct nets +1 total.
func (e *modifiedExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			switch tok {
			case "switch":
				ctx.AddCondition(1)
			case "case":
				ctx.AddCondition(-1)
			}
			if !yield(tok) {
				return
			}
		}
	}
}
