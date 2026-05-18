package chamele

import (
	"io"
	"iter"
)

// ColumnSpec describes one output column. It is the single type used by both
// the built-in tabular formatter and by extensions that add custom columns
// via FunctionInfoColumns.
//
// Header is the (fixed-width, padded) header string for the column. Value is
// the per-function getter; the return type is `any` so extensions can return
// strings or numbers without conversion gymnastics. AvgCaption, when
// non-empty, causes the tabular formatter to include this column in the
// per-file average row.
type ColumnSpec struct {
	Header     string
	AvgCaption string
	Value      func(*FunctionInfo) any
}

// ColumnItem is an alias retained for readability inside output_scheme.go and
// for callers that grew up with the older name.
type ColumnItem = ColumnSpec

// Extension is the interface implemented by each optional metric extension.
// Process is called within the lazy token pipeline — each token passes through
// ALL processors AND state machines before the next token is consumed, so
// ctx.CurrentFunction reflects the function currently being analyzed.
type Extension interface {
	Name() string
	// OrderingIndex controls pipeline position.
	// Negative → before built-in processors; 0+ → after (default 1000).
	OrderingIndex() int
	// Process wraps the token stream. ctx.CurrentFunction is updated by state
	// machines before each token reaches the extension (same token, interleaved).
	Process(tokens iter.Seq[string], ctx *FileInfoBuilder) iter.Seq[string]
	FunctionInfoColumns() []ColumnSpec
}

// CrossFileExtension is optionally implemented by extensions that need a
// post-analysis pass over all FileInformation (e.g. fan-in/out, duplicate detection).
type CrossFileExtension interface {
	Extension
	CrossFileProcess(files []FileInformation) []FileInformation
}

// Printer is optionally implemented by extensions that produce their own summary output.
type Printer interface {
	PrintResult(w io.Writer) error
}

var registeredExtensions []Extension

// RegisterExtension registers a global extension.
func RegisterExtension(ext Extension) {
	registeredExtensions = append(registeredExtensions, ext)
}

// RegisteredExtensions returns all globally registered extensions.
func RegisteredExtensions() []Extension {
	return registeredExtensions
}
