package chamele

import (
	"errors"
	"io"
	"iter"
)

// Phase is when an extension's Process runs relative to the built-in
// processor pipeline (Preprocessing, CommentCounter, LineCounter, TokenCounter,
// ConditionCounter).
type Phase int

const (
	// PhasePreBuiltins runs the extension before the built-in processors,
	// i.e. against the raw tokenizer output. Use this if your extension
	// needs to see whitespace or pre-comment tokens (e.g. ignoreassert,
	// dumpcomments, modified — anything that filters tokens out of the
	// stream before built-ins count them).
	PhasePreBuiltins Phase = -1

	// PhasePostBuiltins runs the extension after the built-in processors,
	// i.e. against the same tokens that drive the language reader's state
	// machine. This is the default.
	PhasePostBuiltins Phase = 0
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
	//
	// Deprecated: use the optional PhaseProvider interface (Phase() Phase) to
	// declare placement explicitly. OrderingIndex is still honoured —
	// negative values map to PhasePreBuiltins, non-negative to PhasePostBuiltins.
	OrderingIndex() int
	// Process wraps the token stream. ctx.CurrentFunction is updated by state
	// machines before each token reaches the extension (same token, interleaved).
	Process(tokens iter.Seq[string], ctx *FileInfoBuilder) iter.Seq[string]
	FunctionInfoColumns() []ColumnSpec
}

// PhaseProvider is an optional Extension interface. If implemented, Phase()
// supersedes OrderingIndex for deciding pipeline placement.
type PhaseProvider interface {
	Phase() Phase
}

// ExtensionPhase returns the Phase for e, preferring an explicit Phase()
// implementation and falling back to OrderingIndex semantics.
func ExtensionPhase(e Extension) Phase {
	if p, ok := e.(PhaseProvider); ok {
		return p.Phase()
	}
	if e.OrderingIndex() < 0 {
		return PhasePreBuiltins
	}
	return PhasePostBuiltins
}

// CrossFileExtension is optionally implemented by extensions that need a
// post-analysis pass over all FileInformation (e.g. fan-in/out, duplicate detection).
type CrossFileExtension interface {
	Extension
	CrossFileProcess(files []FileInformation) []FileInformation
}

// Printer is optionally implemented by extensions that produce their own
// summary output. Printers are invoked by RunPrinters at the end of an
// analysis run, in registration order.
type Printer interface {
	PrintResult(w io.Writer) error
}

// ExtensionFactory builds a fresh Extension instance for one analysis run.
// Use this for extensions that accumulate cross-file state (duplicate,
// boolcount) so that running Analyze twice in the same process produces
// independent metrics.
type ExtensionFactory func() Extension

var registeredFactories []ExtensionFactory

// RegisterExtensionFactory registers a per-run factory. Each call to
// NewRegisteredExtensions invokes every factory once and returns the fresh
// instances.
func RegisterExtensionFactory(f ExtensionFactory) {
	registeredFactories = append(registeredFactories, f)
}

// RegisterExtension registers an Extension. The given instance is reused
// across every Analyze run; this is correct for stateless extensions whose
// per-function state lives entirely on FunctionInfo.Ext.
//
// Extensions that accumulate cross-file or per-run state should call
// RegisterExtensionFactory instead.
func RegisterExtension(ext Extension) {
	RegisterExtensionFactory(func() Extension { return ext })
}

// NewRegisteredExtensions invokes every registered factory once and returns
// the freshly built Extension slice. Call this once per Analyze run.
func NewRegisteredExtensions() []Extension {
	out := make([]Extension, 0, len(registeredFactories))
	for _, f := range registeredFactories {
		out = append(out, f())
	}
	return out
}

// RegisteredExtensions returns a snapshot built by invoking every factory
// once. Kept for callers that don't (yet) thread a per-run extension slice.
//
// Deprecated: prefer NewRegisteredExtensions and pass the result through
// your call stack; calling RegisteredExtensions multiple times in the same
// run will give different instances.
func RegisteredExtensions() []Extension {
	return NewRegisteredExtensions()
}

// RunPrinters invokes PrintResult on every Printer in exts, in slice order.
// Errors from individual printers are joined so one failure doesn't suppress
// the rest. Pass io.Discard to silence Printer output.
func RunPrinters(w io.Writer, exts []Extension) error {
	var errs []error
	for _, e := range exts {
		if p, ok := e.(Printer); ok {
			if err := p.PrintResult(w); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
