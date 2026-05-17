// Package erlang implements the Erlang language reader.
package erlang

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewErlangReader()) }

// ErlangReader handles Erlang source files (.erl).
// Erlang function pattern: `name(Args) -> body.`
// Module attributes start with `-`.
type ErlangReader struct{}

func NewErlangReader() *ErlangReader { return &ErlangReader{} }

func (r *ErlangReader) Extensions() []string    { return []string{"erl"} }
func (r *ErlangReader) LanguageNames() []string { return []string{"erlang"} }

func (r *ErlangReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, `|%%[^\n]*`) // Erlang %% comments
}

func (r *ErlangReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "%") {
		return tok[1:], true
	}
	return "", false
}

func (r *ErlangReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "case": {}, "when": {},
		"and": {}, "or": {},
	}
}

func (r *ErlangReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newErlangMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Erlang state machine ----
// Erlang functions look like: name(Arg1, Arg2) -> Body.
// Module attributes look like: -module(foo). -export([...]).
// We detect functions by: identifier then '(' at top-level (punctuated=false).

type erlangMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	punctuated bool // last non-ws was '-' (module attribute)
	depth      int  // paren depth inside parameter list
	lastToken  string
}

func newErlangMachine(ctx languages.Context) *tokenizer.Machine {
	s := &erlangMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func isAlphaLower(b byte) bool {
	return (b >= 'a' && b <= 'z')
}

func (s *erlangMachine) stateGlobal(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "-":
		s.punctuated = true
	case ".":
		// End of a function clause
		s.punctuated = false
	default:
		if !s.punctuated && len(tok) > 0 && isAlphaLower(tok[0]) {
			// Potential function name
			s.ctx.PushNewFunction(tok)
			s.m.Next(s.stateAfterName)
		}
		s.punctuated = false
	}
	return false
}

func (s *erlangMachine) stateAfterName(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "(" {
		s.depth = 1
		s.m.Next(s.stateParams)
	} else {
		// Not a function; abort
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *erlangMachine) stateParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		s.depth++
	case ")":
		s.depth--
		if s.depth == 0 {
			s.m.Next(s.stateAfterParams)
		}
	default:
		if s.depth == 1 && tok != "," {
			s.ctx.Parameter(tok)
		}
	}
	return false
}

func (s *erlangMachine) stateAfterParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "->":
		// Confirmed function body
		s.m.Next(s.stateBody)
	case "when":
		// Guard clause — skip until ->
		s.m.Next(s.stateGuard)
	default:
		// Not a function definition (e.g. function call)
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *erlangMachine) stateGuard(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "->" {
		s.m.Next(s.stateBody)
	}
	return false
}

func (s *erlangMachine) stateBody(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case ".":
		// End of function
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal)
	case ";":
		// Clause separator — end this function and start a new clause
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal)
	}
	return false
}
