package chamele

import (
	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
)

// Analyze walks paths, analyses every recognised source file, and returns
// one FileInformation per file. Language readers must be registered before
// calling (e.g. import _ "github.com/iszlai/chamele-go/languages/all").
//
// Results are returned in a stable order (by file path) regardless of the
// number of worker goroutines used.
//
// Use AnalyzeWithExtensions instead if you also want the per-run extension
// instances (so you can call RunPrinters on them).
func Analyze(paths []string, opts ...Option) ([]FileInformation, error) {
	files, _, err := AnalyzeWithExtensions(paths, opts...)
	return files, err
}

// AnalyzeWithExtensions is Analyze that also returns the per-run extension
// instances.
func AnalyzeWithExtensions(paths []string, opts ...Option) ([]FileInformation, []Extension, error) {
	e := New(opts...)
	files, err := e.AnalyzePaths(paths)
	return files, e.Extensions(), err
}

// AnalyzeFile analyses a single source file.
func AnalyzeFile(path string, opts ...Option) (*FileInformation, error) {
	e := New(opts...)
	src, err := stringx.ReadFile(path)
	if err != nil {
		return &FileInformation{Filename: path}, err
	}
	r := languages.GetReaderForFilename(path)
	if r == nil {
		return &FileInformation{Filename: path}, nil
	}
	return e.AnalyzeSource(path, src, r), nil
}

// AnalyzeFiles analyses the given list of file paths.
func AnalyzeFiles(paths []string, opts ...Option) ([]FileInformation, error) {
	e := New(opts...)
	return e.analyzeFiles(paths)
}
