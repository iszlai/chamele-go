package chamele

// Analyze analyzes all source files under the given paths and returns one
// FileInformation per file. Language readers must be registered before
// calling (e.g. via import _ "github.com/iszlai/chamele-go/languages/all").
// Implemented in Phase 2.
func Analyze(paths []string, opts ...Option) ([]FileInformation, error) {
	panic("not implemented")
}

// AnalyzeFile analyzes a single source file.
// Implemented in Phase 2.
func AnalyzeFile(path string, opts ...Option) (*FileInformation, error) {
	panic("not implemented")
}

// AnalyzeFiles analyzes the given list of file paths.
// Implemented in Phase 2.
func AnalyzeFiles(paths []string, opts ...Option) ([]FileInformation, error) {
	panic("not implemented")
}
