// Package objc implements the Objective-C language reader.
package objc

import (
	"iter"

	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewObjCReader()) }

// ObjCReader handles Objective-C source files (.m).
// It uses the standard C-like state machines; the ObjC method syntax
// (+/-) is not separately tracked at this level of complexity analysis.
type ObjCReader struct {
	*clike.CLikeReader
}

func NewObjCReader() *ObjCReader {
	return &ObjCReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *ObjCReader) Extensions() []string    { return []string{"m"} }
func (r *ObjCReader) LanguageNames() []string { return []string{"objectivec", "objective-c", "objc"} }

func (r *ObjCReader) Tokenize(src []byte) iter.Seq[string] {
	return r.CLikeReader.Tokenize(src)
}

// RunTokens uses the standard C-like parallel machines.
func (r *ObjCReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	clike.RunParallel(tokens, ctx,
		clike.NewCLikeStates,
		clike.NewCLikeNestingStackStates,
	)
}
