// Package languages provides the Reader interface and the language registry.
// Language readers register themselves via init() functions in their
// respective sub-packages.
package languages

import "iter"

// Reader is the interface implemented by each language reader.
// It is a stable public API from v0.1.
type Reader interface {
	// Extensions returns the file extensions this reader handles (without dot).
	Extensions() []string
	// LanguageNames returns the canonical names for this language (e.g. {"cpp","c"}).
	LanguageNames() []string
	// Tokenize returns a token sequence for the given source bytes.
	Tokenize(src []byte) iter.Seq[string]
}

// Context is the analysis context passed to language state machines.
// It is satisfied by *chamele.FileInfoBuilder and defines the subset of
// FileInfoBuilder methods that language readers may call.
type Context interface {
	TryNewFunction(name string)
	ConfirmNewFunction()
	RestartNewFunction(name string)
	PushNewFunction(name string)
	AddCondition(inc int)
	AddToFunctionName(app string)
	AddToLongFunctionName(app string)
	Parameter(tok string)
	PopNesting()
	AddBareNesting()
	AddNamespace(name string)
	CurrentNestingLevel() int
	WithNamespace(name string) string
	CurrentFunctionLongName() string
	// EndOfFunction finalises the current function and pops stacked_functions.
	// Called by language readers that manage their own function-body nesting
	// (e.g. Go) rather than relying on CLikeNestingStackStates.
	EndOfFunction()
	// HasStackedFunction reports whether the function stack is non-empty and
	// the outermost stacked function is a real function (not *global*).
	// IsInsideFunction reports whether the current scope is inside a real
	// (non-global) function. Used by the Go reader to distinguish a method
	// receiver `(recv Type)` from a closure parameter list.
	IsInsideFunction() bool
	// AddNLOC adjusts the NLOC counters by count (may be negative for docstrings).
	AddNLOC(count int)
	// CurrentFunctionName returns the name of the current function (may be "*global*").
	CurrentFunctionName() string
}

// TokenRunner is optionally implemented by readers that drive parallel state
// machines over the processed token stream. RunTokens is called after all
// standard processors have run.
type TokenRunner interface {
	RunTokens(tokens iter.Seq[string], ctx Context)
}

var registry []Reader

// Register adds a reader to the global registry. Call from init().
func Register(r Reader) { registry = append(registry, r) }

// GetReaderForFilename returns the reader whose extension list matches the
// file's extension, or nil if none matches.
func GetReaderForFilename(path string) Reader {
	ext := extension(path)
	for _, r := range registry {
		for _, e := range r.Extensions() {
			if e == ext {
				return r
			}
		}
	}
	return nil
}

// Get returns the first reader whose LanguageNames contains the given name.
func Get(name string) Reader {
	for _, r := range registry {
		for _, n := range r.LanguageNames() {
			if n == name {
				return r
			}
		}
	}
	return nil
}

func extension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i+1:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}
