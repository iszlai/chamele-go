// Package dumpcomments is a debug aid that prints comments to stderr.
package dumpcomments

import (
	"fmt"
	"iter"
	"os"
	"strings"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type dumpCommentsExt struct{}

func New() chamele.Extension { return &dumpCommentsExt{} }

func (e *dumpCommentsExt) Name() string        { return "dumpcomments" }
func (e *dumpCommentsExt) OrderingIndex() int  { return 1000 }
func (e *dumpCommentsExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

func (e *dumpCommentsExt) Process(tokens iter.Seq[string], _ *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if strings.HasPrefix(tok, "//") || strings.HasPrefix(tok, "/*") {
				fmt.Fprintln(os.Stderr, tok)
			}
			if !yield(tok) {
				return
			}
		}
	}
}
