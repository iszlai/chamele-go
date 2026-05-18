package chamele

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
	"golang.org/x/sync/errgroup"
)

// Engine is the single entry point for chamele analysis. It owns the
// processor pipeline, the per-run extension instances, and the configured
// options. One Engine per analysis run — re-instantiate to clear extension
// state.
type Engine struct {
	processors []Processor
	exts       []Extension
	opts       Options
}

// New builds an Engine with the given options.
func New(opts ...Option) *Engine {
	o := applyOptions(opts)
	exts := o.Extensions
	if exts == nil {
		exts = NewRegisteredExtensions()
	}
	return &Engine{
		processors: DefaultProcessors(),
		exts:       exts,
		opts:       o,
	}
}

// Extensions returns the per-run extension instances. Pass these to
// RunPrinters to emit per-extension summary output.
func (e *Engine) Extensions() []Extension { return e.exts }

// Options returns the engine's configured options.
func (e *Engine) Options() Options { return e.opts }

// AnalyzePaths walks paths, analyses every recognised source file, and
// returns the per-file results.
func (e *Engine) AnalyzePaths(paths []string) ([]FileInformation, error) {
	files := sourceFiles(paths, e.opts)
	return e.analyzeFiles(files)
}

// AnalyzeFile analyses a single source file. Returns an empty
// FileInformation (just Filename set) on read/parse failure so the result
// slice still has the expected length.
func (e *Engine) AnalyzeFile(path string) *FileInformation {
	src, err := stringx.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Fail to read source file '%s'\n", path)
		return &FileInformation{Filename: path}
	}
	r := languages.GetReaderForFilename(path)
	if r == nil {
		return &FileInformation{Filename: path}
	}
	return e.AnalyzeSource(path, src, r)
}

// AnalyzeSource runs the pipeline over pre-loaded source bytes.
func (e *Engine) AnalyzeSource(filename string, src []byte, r languages.Reader) *FileInformation {
	ctx := NewFileInfoBuilder(filename)
	tokens := r.Tokenize(src)

	var pre, post []Extension
	for _, ext := range e.exts {
		if ExtensionPhase(ext) == PhasePreBuiltins {
			pre = append(pre, ext)
		} else {
			post = append(post, ext)
		}
	}

	for _, ext := range pre {
		tokens = ext.Process(tokens, ctx)
	}
	for _, proc := range e.processors {
		tokens = proc(tokens, ctx, r)
	}
	for _, ext := range post {
		tokens = ext.Process(tokens, ctx)
	}

	if tr, ok := r.(languages.TokenRunner); ok {
		tr.RunTokens(tokens, ctx)
	} else {
		for range tokens {
		}
	}

	return ctx.Build()
}

// analyzeFiles drives the worker pool over an already-resolved file list.
// Workers pull indices from a shared atomic counter (no channel, no
// per-job allocation).
func (e *Engine) analyzeFiles(paths []string) ([]FileInformation, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	threads := e.opts.Threads
	if threads <= 0 {
		threads = runtime.NumCPU()
	}
	if threads > len(paths) {
		threads = len(paths)
	}

	results := make([]FileInformation, len(paths))
	var idx atomic.Int64
	total := int64(len(paths))

	var g errgroup.Group
	for range threads {
		g.Go(func() error {
			for {
				i := idx.Add(1) - 1
				if i >= total {
					return nil
				}
				fi := e.AnalyzeFile(paths[i])
				results[i] = *fi
			}
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	for _, ext := range e.exts {
		if cfe, ok := ext.(CrossFileExtension); ok {
			results = cfe.CrossFileProcess(results)
		}
	}
	return results, nil
}
