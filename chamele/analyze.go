package chamele

import (
	"runtime"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
	"golang.org/x/sync/errgroup"
)

// Analyze walks paths, analyses every recognised source file, and returns one
// FileInformation per file. Language readers must be registered before calling
// (e.g. import _ "github.com/iszlai/chamele-go/languages/all").
//
// Results are returned in a stable order (by file path) regardless of the
// number of worker goroutines used.
func Analyze(paths []string, opts ...Option) ([]FileInformation, error) {
	files, _, err := AnalyzeWithExtensions(paths, opts...)
	return files, err
}

// AnalyzeWithExtensions is Analyze that also returns the per-run extension
// instances. Use the returned slice with RunPrinters to emit per-extension
// summaries (e.g. duplicate-block listings, bool-count rate) at the end of
// your output.
func AnalyzeWithExtensions(paths []string, opts ...Option) ([]FileInformation, []Extension, error) {
	o := applyOptions(opts)
	files := sourceFiles(paths, o)
	exts := NewRegisteredExtensions()
	results, err := analyzeFiles(files, o, exts)
	return results, exts, err
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
	exts := NewRegisteredExtensions()
	return analyzeFiles(paths, o, exts)
}

// analyzeFiles runs the FileAnalyzer over each path using a worker pool.
// The default pool size is runtime.NumCPU(). All workers share the same
// extension slice so per-run state (e.g. duplicate detection accumulators,
// cross-file fan-in maps) sees every file from the same instance.
func analyzeFiles(paths []string, opts Options, exts []Extension) ([]FileInformation, error) {
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

	analyzer := NewFileAnalyzerWithExts(exts)
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

	// Cross-file extensions (fan-in/fan-out, duplicate detection) run after
	// every file has been analysed, on the main goroutine, so they see
	// deterministic order. They use the same instances that ran Process so
	// any accumulators they built during the per-file pass are visible.
	for _, ext := range exts {
		if cfe, ok := ext.(CrossFileExtension); ok {
			results = cfe.CrossFileProcess(results)
		}
	}
	return results, nil
}
