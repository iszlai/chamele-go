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
	// 'when' in function guards is not counted (matches Python lizard parity).
	return map[string]struct{}{
		"if": {}, "case": {},
		"and": {}, "or": {},
		"?": {}, // macro expansion adds to complexity
	}
}

func (r *ErlangReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newErlangMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// blockKind distinguishes if/case/begin/receive blocks from fun expressions.
type blockKind int

const (
	blockGeneral blockKind = iota // if/case/begin/receive — no new function
	blockFun                      // fun ... end — stacks a new "fun" function
)

// ---- Erlang state machine ----
// Functions: identifier(Args) -> Body. or identifier(Args) -> Body;
// Anonymous funs: fun (Args) -> Body end
// Module attributes: -module(...). -export([...]).

type erlangMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	punctuated bool        // last non-ws was '-' (module attribute)
	depth      int         // paren depth in outer function params
	funDepth   int         // paren depth in fun params
	blockStack []blockKind // nesting inside function body
	lastToken  string
}

func newErlangMachine(ctx languages.Context) *tokenizer.Machine {
	s := &erlangMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func isAlphaLower(b byte) bool {
	return b >= 'a' && b <= 'z'
}

func (s *erlangMachine) stateGlobal(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "-":
		s.punctuated = true
	case ".":
		s.punctuated = false
	default:
		if !s.punctuated && len(tok) > 0 && isAlphaLower(tok[0]) {
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
		// Not a function — abandon and re-process token
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
		s.blockStack = s.blockStack[:0] // reset for this clause
		s.m.Next(s.stateBody)
	case "when":
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
		s.blockStack = s.blockStack[:0]
		s.m.Next(s.stateBody)
	}
	return false
}

func (s *erlangMachine) stateBody(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "if", "case", "begin", "receive":
		s.blockStack = append(s.blockStack, blockGeneral)
	case "fun":
		// Could be fun (...) -> body end (anonymous fn) or fun mod:name/arity (reference).
		// We peek at the next token in stateFunOrRef.
		s.m.Next(s.stateFunOrRef)
	case "end":
		if len(s.blockStack) > 0 {
			top := s.blockStack[len(s.blockStack)-1]
			s.blockStack = s.blockStack[:len(s.blockStack)-1]
			if top == blockFun {
				s.ctx.EndOfFunction()
			}
		}
	case ".":
		if len(s.blockStack) == 0 {
			s.ctx.EndOfFunction()
			s.m.Next(s.stateGlobal)
		}
	case ";":
		if len(s.blockStack) == 0 {
			// Function clause separator — end this clause, start next
			s.ctx.EndOfFunction()
			s.m.Next(s.stateGlobal)
		}
	}
	return false
}

// stateFunOrRef determines if `fun` is an anonymous function (fun (...) -> end)
// or a function reference (fun mod:name/arity). Only anonymous funs get a new function.
func (s *erlangMachine) stateFunOrRef(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "(" {
		// Anonymous fun: fun (Args) -> Body end
		s.ctx.PushNewFunction("fun")
		s.blockStack = append(s.blockStack, blockFun)
		s.funDepth = 1
		s.m.Next(s.stateFunParams)
	} else {
		// Function reference: fun mod:name/arity or fun name/arity — no new function
		s.m.Next(s.stateBody, tok)
	}
	return false
}

func (s *erlangMachine) stateFunParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		s.funDepth++
	case ")":
		s.funDepth--
		if s.funDepth == 0 {
			s.m.Next(s.stateFunAfterParams)
		}
	default:
		if s.funDepth == 1 && tok != "," {
			s.ctx.Parameter(tok)
		}
	}
	return false
}

func (s *erlangMachine) stateFunAfterParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "->":
		s.m.Next(s.stateBody)
	case "when":
		s.m.Next(s.stateFunGuard)
	}
	return false
}

func (s *erlangMachine) stateFunGuard(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "->" {
		s.m.Next(s.stateBody)
	}
	return false
}
