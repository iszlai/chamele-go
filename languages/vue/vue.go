// Package vue implements the Vue.js language reader.
package vue

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/javascript"
)

func init() { languages.Register(NewVueReader()) }

// VueReader handles Vue.js single-file components (.vue).
// It delegates to the JS reader for all function detection, but the preprocessor
// only emits tokens that appear inside <script> ... </script> blocks.
type VueReader struct {
	*javascript.JSReader
}

func NewVueReader() *VueReader {
	return &VueReader{JSReader: javascript.NewJSReader()}
}

func (r *VueReader) Extensions() []string    { return []string{"vue"} }
func (r *VueReader) LanguageNames() []string { return []string{"vue", "vuejs"} }

func (r *VueReader) Tokenize(src []byte) iter.Seq[string] {
	return r.JSReader.Tokenize(src)
}

// Preprocess filters tokens to only those inside <script> ... </script>.
func (r *VueReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		inScript := false
		for tok := range tokens {
			lower := strings.ToLower(tok)
			if strings.HasPrefix(lower, "<script") {
				inScript = true
				continue
			}
			if strings.HasPrefix(lower, "</script") {
				inScript = false
				continue
			}
			if !inScript {
				continue
			}
			if tok == "\n" || !isHSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}

// RunTokens inherits the JS machine.
func (r *VueReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	r.JSReader.RunTokens(tokens, ctx)
}

func isHSpace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' {
			return false
		}
	}
	return len(s) > 0
}
