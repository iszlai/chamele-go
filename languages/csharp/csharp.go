// Package csharp implements the C# language reader.
package csharp

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewCSharpReader()) }

// CSharpReader handles C# source files. It extends CLikeReader with the ??
// null-coalescing operator token and includes it in the condition set.
type CSharpReader struct {
	*clike.CLikeReader
}

func NewCSharpReader() *CSharpReader {
	return &CSharpReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *CSharpReader) Extensions() []string    { return []string{"cs"} }
func (r *CSharpReader) LanguageNames() []string { return []string{"csharp"} }

func (r *CSharpReader) Tokenize(src []byte) iter.Seq[string] {
	// Add C# null-coalescing operator ?? to the tokenizer.
	return tokenizer.GenerateTokens(src, clike.CLikeAddition()+`|(?:\?\?)`)
}

// GetConditions returns C# CCN conditions including ?? (null-coalescing).
func (r *CSharpReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {},
		"&&": {}, "||": {}, "case": {}, "?": {}, "??": {},
	}
}

// RunTokens uses the standard C-like parallel machines.
func (r *CSharpReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	clike.RunParallel(tokens, ctx,
		clike.NewCLikeStates,
		clike.NewCLikeNestingStackStates,
	)
}
