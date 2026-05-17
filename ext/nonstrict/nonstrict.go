// Package nonstrict is a marker extension that silences all other extension output.
package nonstrict

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type nonstrictExt struct{}

func New() chamele.Extension { return &nonstrictExt{} }

func (e *nonstrictExt) Name() string                              { return "nonstrict" }
func (e *nonstrictExt) OrderingIndex() int                        { return 1000 }
func (e *nonstrictExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

func (e *nonstrictExt) Process(tokens iter.Seq[string], _ *chamele.FileInfoBuilder) iter.Seq[string] {
	return tokens
}
