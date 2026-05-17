// Package ttcn implements the TTCN-3 language reader.
package ttcn

import (
	"iter"

	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewTTCNReader()) }

// TTCNReader handles TTCN-3 source files (.ttcn, .ttcn3, .ttcnpp).
// TTCN-3 uses `function name(params) { ... }` and `testcase name(params) { ... }`.
// It is C-like, so we can extend the CLikeReader.
type TTCNReader struct {
	*clike.CLikeReader
}

func NewTTCNReader() *TTCNReader {
	return &TTCNReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *TTCNReader) Extensions() []string    { return []string{"ttcn", "ttcn3", "ttcnpp"} }
func (r *TTCNReader) LanguageNames() []string { return []string{"ttcn", "ttcn3"} }

func (r *TTCNReader) Tokenize(src []byte) iter.Seq[string] {
	return r.CLikeReader.Tokenize(src)
}

// GetConditions returns TTCN-3 CCN conditions.
func (r *TTCNReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "alt": {}, "altstep": {}, "interleave": {}, "goto": {},
		"and": {}, "or": {}, "xor": {}, "case": {},
	}
}

// RunTokens uses the TTCN-specific state machine that recognises
// `function`, `testcase`, and `control` keywords.
func (r *TTCNReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	clike.RunParallel(tokens, ctx,
		newTTCNMachineConstructor,
		clike.NewCLikeNestingStackStates,
	)
}
