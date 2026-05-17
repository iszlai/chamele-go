package chamele

import (
	"fmt"
	"iter"
	"os"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
)

// parallelRunner is implemented by language readers that drive parallel state
// machines over the processed token stream (Phase 3+).
type parallelRunner interface {
	RunTokens(tokens iter.Seq[string], ctx *FileInfoBuilder)
}

// FileAnalyzer runs the processor pipeline over a single source file and
// returns the resulting FileInformation.
type FileAnalyzer struct {
	processors []Processor
}

// NewFileAnalyzer creates an analyzer with the standard processor pipeline.
func NewFileAnalyzer() *FileAnalyzer {
	return &FileAnalyzer{processors: DefaultProcessors()}
}

// NewFileAnalyzerWithExts creates an analyzer whose pipeline includes the
// given extensions inserted at their declared ordering index.
func NewFileAnalyzerWithExts(exts []Extension) *FileAnalyzer {
	procs := DefaultProcessors()
	_ = exts // extension integration wired in Phase 6
	return &FileAnalyzer{processors: procs}
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

	for _, proc := range a.processors {
		tokens = proc(tokens, ctx, r)
	}

	// Feed processed tokens into the reader's parallel state machines (if any).
	// For readers that don't implement parallelRunner (Phase 1/2 stubs), just drain.
	if pr, ok := r.(parallelRunner); ok {
		pr.RunTokens(tokens, ctx)
	} else {
		for range tokens {
		}
	}

	return ctx.Build()
}
