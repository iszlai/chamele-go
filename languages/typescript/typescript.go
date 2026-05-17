// Package typescript implements the TypeScript language reader.
package typescript

import (
	"iter"

	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/javascript"
)

func init() { languages.Register(NewTSReader()) }

// TSReader handles TypeScript source files (.ts).
// TypeScript is a superset of JavaScript so we delegate entirely to the JS reader
// state machine, which already handles arrow functions, named functions, and
// class methods. The only overrides are Extensions and LanguageNames.
type TSReader struct {
	*javascript.JSReader
}

func NewTSReader() *TSReader {
	return &TSReader{JSReader: javascript.NewJSReader()}
}

func (r *TSReader) Extensions() []string    { return []string{"ts"} }
func (r *TSReader) LanguageNames() []string { return []string{"typescript", "ts"} }

func (r *TSReader) Tokenize(src []byte) iter.Seq[string] {
	return r.JSReader.Tokenize(src)
}

// RunTokens inherits the JS machine which handles the common function patterns.
func (r *TSReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	r.JSReader.RunTokens(tokens, ctx)
}
