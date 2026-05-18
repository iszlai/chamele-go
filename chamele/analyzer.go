package chamele

import (
	"fmt"
	"os"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
)

// FileAnalyzer runs the processor pipeline over a single source file and
// returns the resulting FileInformation.
type FileAnalyzer struct {
	processors []Processor
	exts       []Extension
}

// NewFileAnalyzer creates an analyzer with the standard processor pipeline
// and every globally registered extension.
func NewFileAnalyzer() *FileAnalyzer {
	return NewFileAnalyzerWithExts(RegisteredExtensions())
}

// NewFileAnalyzerWithExts creates an analyzer whose pipeline includes the
// given extensions inserted at their declared ordering index.
func NewFileAnalyzerWithExts(exts []Extension) *FileAnalyzer {
	return &FileAnalyzer{processors: DefaultProcessors(), exts: exts}
}

// AnalyzeFile reads path from disk, selects a reader, and analyses it.
// Returns an empty FileInformation (with just Filename set) on read/parse errors.
func (a *FileAnalyzer) AnalyzeFile(path string) *FileInformation {
	src, err := stringx.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Fail to read source file '%s'\n", path)
		return &FileInformation{Filename: path}
	}
	r := languages.GetReaderForFilename(path)
	if r == nil {
		return &FileInformation{Filename: path}
	}
	return a.AnalyzeSourceCode(path, src, r)
}

// AnalyzeSourceCode runs the pipeline over pre-loaded source bytes.
func (a *FileAnalyzer) AnalyzeSourceCode(filename string, src []byte, r languages.Reader) *FileInformation {
	ctx := NewFileInfoBuilder(filename)
	tokens := r.Tokenize(src)

	// Split extensions by phase.
	var pre, post []Extension
	for _, e := range a.exts {
		if ExtensionPhase(e) == PhasePreBuiltins {
			pre = append(pre, e)
		} else {
			post = append(post, e)
		}
	}

	for _, e := range pre {
		tokens = e.Process(tokens, ctx)
	}
	for _, proc := range a.processors {
		tokens = proc(tokens, ctx, r)
	}
	for _, e := range post {
		tokens = e.Process(tokens, ctx)
	}

	// Feed processed tokens into the reader's parallel state machines (if any).
	// For readers that don't implement TokenRunner (Phase 1/2 stubs), just drain.
	if tr, ok := r.(languages.TokenRunner); ok {
		tr.RunTokens(tokens, ctx)
	} else {
		for range tokens {
		}
	}

	return ctx.Build()
}
