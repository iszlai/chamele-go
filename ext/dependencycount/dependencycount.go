// Package dependencycount implements the dependencycount extension (stub — full implementation Phase 6 completion).
package dependencycount

import (
	"iter"
	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string        { return "dependencycount" }
func (e *ext) OrderingIndex() int  { return 1000 }
func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec { return nil }
func (e *ext) Process(tokens iter.Seq[string], _ *chamele.FileInfoBuilder) iter.Seq[string] {
	return tokens
}
