package chamele

import (
	"github.com/iszlai/chamele-go/languages"
)

// FileAnalyzer is a back-compat wrapper around Engine. New code should use
// chamele.New(...) and its AnalyzeFile/AnalyzeSource methods; FileAnalyzer
// remains because it appears in many language- and extension-level tests.
type FileAnalyzer struct {
	eng *Engine
}

// NewFileAnalyzer creates an analyzer with the standard processor pipeline
// and every globally registered extension.
func NewFileAnalyzer() *FileAnalyzer {
	return &FileAnalyzer{eng: New()}
}

// NewFileAnalyzerWithExts creates an analyzer whose pipeline includes the
// given extensions, instead of the global registry.
func NewFileAnalyzerWithExts(exts []Extension) *FileAnalyzer {
	return &FileAnalyzer{eng: New(WithExtensions(exts...))}
}

// AnalyzeFile reads path from disk, selects a reader, and analyses it.
func (a *FileAnalyzer) AnalyzeFile(path string) *FileInformation {
	return a.eng.AnalyzeFile(path)
}

// AnalyzeSourceCode runs the pipeline over pre-loaded source bytes.
func (a *FileAnalyzer) AnalyzeSourceCode(filename string, src []byte, r languages.Reader) *FileInformation {
	return a.eng.AnalyzeSource(filename, src, r)
}
