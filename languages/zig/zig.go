// Package zig implements the Zig language reader.
package zig

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewZigReader()) }

// ZigReader handles Zig source files (.zig).
// Zig uses `fn` as the function keyword with Go-like brace-depth tracking.
type ZigReader struct{}

func NewZigReader() *ZigReader { return &ZigReader{} }

func (r *ZigReader) Extensions() []string    { return []string{"zig"} }
func (r *ZigReader) LanguageNames() []string { return []string{"zig"} }

func (r *ZigReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, "")
}

func (r *ZigReader) GetComment(tok string) (string, bool) {
	if len(tok) >= 2 && (tok[:2] == "//" || tok[:2] == "/*") {
		return tok[2:], true
	}
	return "", false
}

func (r *ZigReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {},
		"and": {}, "or": {}, "orelse": {},
	}
}

func (r *ZigReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newZigMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Zig state machine ----

type zigMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	braceDepth int
	funcDepths []int
}

func newZigMachine(ctx languages.Context) *tokenizer.Machine {
	s := &zigMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *zigMachine) stateGlobal(tok string) bool {
	switch tok {
	case "fn":
		s.ctx.PushNewFunction("")
		s.m.Next(s.stateFunctionName)
	case "{":
		s.braceDepth++
	case "}":
		s.braceDepth--
		if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
			s.ctx.EndOfFunction()
			s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
		}
	}
	return false
}

func (s *zigMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	default:
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

func (s *zigMachine) stateExpectFunctionDec(tok string) bool {
	if tok == "(" {
		s.m.Next(s.stateFunctionDec(), tok)
	} else {
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *zigMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
	})
}

func (s *zigMachine) stateExpectFunctionImpl(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImpl, "{")
	}
	return false
}

func (s *zigMachine) stateEnteringImpl(_ string) bool {
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}
