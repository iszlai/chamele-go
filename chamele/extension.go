package chamele

import (
	"io"
	"iter"
)

// ColumnSpec describes an additional output column added by an extension.
type ColumnSpec struct {
	Header string
	Value  func(*FunctionInfo) any
}

// Extension is the interface implemented by each optional metric extension.
// Implemented in Phase 6.
type Extension interface {
	Name() string
	OrderingIndex() int
	Process(tokens iter.Seq[string]) iter.Seq[string]
	FunctionInfoColumns() []ColumnSpec
	PrintResult(w io.Writer) error
}

var registeredExtensions []Extension

// RegisterExtension registers a global extension.
func RegisterExtension(ext Extension) {
	registeredExtensions = append(registeredExtensions, ext)
}
