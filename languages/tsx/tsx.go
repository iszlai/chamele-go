// Package tsx implements the TSX/JSX language reader.
package tsx

import (
	"iter"

	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/javascript"
)

func init() { languages.Register(NewTSXReader()) }

// TSXReader handles TSX and JSX source files (.tsx, .jsx).
// Like TypeScript, TSX is a JavaScript superset so we delegate to the JS reader.
type TSXReader struct {
	*javascript.JSReader
}

func NewTSXReader() *TSXReader {
	return &TSXReader{JSReader: javascript.NewJSReader()}
}

func (r *TSXReader) Extensions() []string    { return []string{"tsx", "jsx"} }
func (r *TSXReader) LanguageNames() []string { return []string{"tsx", "jsx"} }

func (r *TSXReader) Tokenize(src []byte) iter.Seq[string] {
	return r.JSReader.Tokenize(src)
}

// RunTokens inherits the JS machine.
func (r *TSXReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	r.JSReader.RunTokens(tokens, ctx)
}
