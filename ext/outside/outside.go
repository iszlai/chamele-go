// Package outside treats the global scope as a named function "*global*".
// When enabled, file-level code is attributed to a function called "*global*".
package outside

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type outsideExt struct{}

func New() chamele.Extension { return &outsideExt{} }

func (e *outsideExt) Name() string        { return "outside" }
func (e *outsideExt) OrderingIndex() int  { return 1000 }
func (e *outsideExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

// Process is a no-op; the *global* function is already tracked internally.
// Enabling this extension causes Build() to include it in the output.
func (e *outsideExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	ctx.IncludeGlobal = true
	return tokens
}
