package chamele

// Options configures an analysis run. Build via the functional options
// (WithLanguages, WithThreads, WithExclude, WithExtensions) or via the
// Engine constructor directly.
type Options struct {
	Languages  []string
	Threads    int
	Exclude    []string
	Extensions []Extension
}

// Option is a functional option for Options.
type Option func(*Options)

// WithLanguages restricts analysis to the named languages.
func WithLanguages(langs ...string) Option {
	return func(o *Options) { o.Languages = langs }
}

// WithThreads sets the number of parallel workers. Zero or negative
// values mean "runtime.NumCPU()".
func WithThreads(n int) Option {
	return func(o *Options) { o.Threads = n }
}

// WithExclude adds fnmatch exclude patterns. Patterns are matched against
// both the full path and the base name of every candidate source file.
func WithExclude(patterns ...string) Option {
	return func(o *Options) { o.Exclude = append(o.Exclude, patterns...) }
}

// WithExtensions overrides the global extension registry. Useful for
// library callers who want to test or run a focused subset of extensions
// without importing ext/all.
func WithExtensions(exts ...Extension) Option {
	return func(o *Options) { o.Extensions = exts }
}

func applyOptions(opts []Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
