package chamele

import (
	"runtime"

	"golang.org/x/sync/errgroup"
	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
)

// Analyze walks paths, analyses every recognised source file, and returns one
// FileInformation per file. Language readers must be registered before calling
// (e.g. import _ "github.com/iszlai/chamele-go/languages/all").
//
// Results are returned in a stable order (by file path) regardless of the
// number of worker goroutines used.
func Analyze(paths []string, opts ...Option) ([]FileInformation, error) {
	o := applyOptions(opts)
	files := sourceFiles(paths, o)
	return analyzeFiles(files, o)
}


// AnalyzeFile analyses a single source file.
func AnalyzeFile(path string, opts ...Option) (*FileInformation, error) {
	_ = applyOptions(opts) // opts reserved for future use (extensions, etc.)
	src, err := stringx.ReadFile(path)
	if err != nil {
		return &FileInformation{Filename: path}, err
	}
	r := languages.GetReaderForFilename(path)
	if r == nil {
		return &FileInformation{Filename: path}, nil
	}
	a := NewFileAnalyzer()
	fi := a.AnalyzeSourceCode(path, src, r)
	return fi, nil
}

// AnalyzeFiles analyses the given list of file paths.
func AnalyzeFiles(paths []string, opts ...Option) ([]FileInformation, error) {
	o := applyOptions(opts)
	return analyzeFiles(paths, o)
}

// analyzeFiles runs the FileAnalyzer over each path using a worker pool.
// The default pool size is runtime.NumCPU().
func analyzeFiles(paths []string, opts Options) ([]FileInformation, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	threads := opts.Threads
	if threads <= 0 {
		threads = runtime.NumCPU()
	}

	results := make([]FileInformation, len(paths))
	jobs := make(chan int, len(paths))
	for i := range paths {
		jobs <- i
	}
	close(jobs)

	analyzer := NewFileAnalyzer()
	var g errgroup.Group
	workers := threads
	if workers > len(paths) {
		workers = len(paths)
	}
	for range workers {
		g.Go(func() error {
			for idx := range jobs {
				fi := analyzer.AnalyzeFile(paths[idx])
				results[idx] = *fi
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
