// Package gdscript implements the GDScript language reader.
package gdscript

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/indented"
)

func init() { languages.Register(NewGDScriptReader()) }

// GDScriptReader handles GDScript source files (.gd).
// GDScript is Python-like: `func name():` introduces a function, and indentation
// determines block scope. The `#` character starts comments.
type GDScriptReader struct{}

func NewGDScriptReader() *GDScriptReader { return &GDScriptReader{} }

func (r *GDScriptReader) Extensions() []string    { return []string{"gd"} }
func (r *GDScriptReader) LanguageNames() []string { return []string{"gdscript", "GDScript"} }

func (r *GDScriptReader) Tokenize(src []byte) iter.Seq[string] {
	untilEnd := `(?:\\\n|[^\n])*`
	return tokenizer.GenerateTokens(src, `|#`+untilEnd)
}

func (r *GDScriptReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "#") {
		return strings.TrimSpace(tok[1:]), true
	}
	return "", false
}

func (r *GDScriptReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elif": {}, "for": {}, "while": {}, "case": {},
		"&&": {}, "||": {}, "and": {}, "or": {},
	}
}

// Preprocess handles GDScript indentation-based nesting (same as Python).
func (r *GDScriptReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		indents := &indented.Indents{Ctx: ctx}
		currentSpaces := 0
		readingLeadingSpace := true

		for tok := range tokens {
			if tok != "\n" {
				if readingLeadingSpace {
					if stringx.IsHSpace(tok) {
						currentSpaces += indented.CountSpaces(tok)
					} else {
						if !strings.HasPrefix(tok, "#") {
							name := ctx.CurrentFunctionName()
							lname := ctx.CurrentFunctionLongName()
							if name == "*global*" || strings.HasSuffix(lname, ")") {
								indents.SetNesting(currentSpaces, tok)
							}
						}
						readingLeadingSpace = false
						if !yield(tok) {
							return
						}
					}
				} else {
					if !stringx.IsHSpace(tok) {
						if !yield(tok) {
							return
						}
					}
				}
			} else {
				readingLeadingSpace = true
				currentSpaces = 0
				if !yield(tok) {
					return
				}
			}
		}
		indents.SetNesting(0, "")
	}
}

func (r *GDScriptReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newGDScriptMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- GDScript state machine ----

type gdScriptMachine struct {
	m   *tokenizer.Machine
	ctx languages.Context
}

func newGDScriptMachine(ctx languages.Context) *tokenizer.Machine {
	s := &gdScriptMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *gdScriptMachine) stateGlobal(tok string) bool {
	if tok == "func" {
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func (s *gdScriptMachine) stateFunctionName(tok string) bool {
	if tok != "(" {
		s.ctx.RestartNewFunction(tok)
		s.ctx.AddToLongFunctionName("(")
	} else {
		s.m.Next(s.stateDec)
	}
	return false
}

func (s *gdScriptMachine) stateDec(tok string) bool {
	switch tok {
	case ")":
		s.ctx.AddToLongFunctionName(" )")
		s.m.Next(s.stateColon)
	default:
		if !stringx.IsHSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
		}
		s.ctx.AddToLongFunctionName(" " + tok)
	}
	return false
}

func (s *gdScriptMachine) stateColon(tok string) bool {
	if tok == ":" {
		// GDScript confirms function at colon (like Python)
		s.m.Next(s.stateGlobal)
	} else {
		s.m.Next(s.stateGlobal)
	}
	return false
}
