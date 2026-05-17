// Package boolcount counts `bool` type occurrences per file.
package boolcount

import (
	"fmt"
	"io"
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type boolCountExt struct {
	totalBool  int
	totalToken int
}

func New() chamele.Extension { return &boolCountExt{} }

func (e *boolCountExt) Name() string        { return "boolcount" }
func (e *boolCountExt) OrderingIndex() int  { return 1000 }
func (e *boolCountExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

func (e *boolCountExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "bool" || tok == "Bool" || tok == "boolean" {
				e.totalBool++
			}
			e.totalToken++
			if !yield(tok) {
				return
			}
		}
	}
}

func (e *boolCountExt) PrintResult(w io.Writer) error {
	denom := e.totalToken
	if denom == 0 {
		denom = 1
	}
	fmt.Fprintf(w, "Total non-comment token: %d\n", e.totalToken)
	fmt.Fprintf(w, "Total bool count: %d\n", e.totalBool)
	fmt.Fprintf(w, "rate %%: %.2f\n", float64(e.totalBool)*100.0/float64(denom))
	return nil
}

// Implement Printer interface.
var _ chamele.Printer = (*boolCountExt)(nil)
