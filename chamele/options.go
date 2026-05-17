package chamele

// Options configures an analysis run.
type Options struct {
	Languages      []string
	Threads        int
	Exclude        []string
	IgnoreWarnings int
	Whitelist      string
}

// Option is a functional option for Options.
type Option func(*Options)

// WithLanguages restricts analysis to the named languages.
func WithLanguages(langs ...string) Option {
	return func(o *Options) { o.Languages = langs }
}

// WithThreads sets the number of parallel workers.
func WithThreads(n int) Option {
	return func(o *Options) { o.Threads = n }
}

// WithExclude adds fnmatch exclude patterns.
func WithExclude(patterns ...string) Option {
	return func(o *Options) { o.Exclude = append(o.Exclude, patterns...) }
}

// WithWhitelist sets the path to the whitelist file.
func WithWhitelist(path string) Option {
	return func(o *Options) { o.Whitelist = path }
}

func applyOptions(opts []Option) Options {
	o := Options{IgnoreWarnings: -1}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
