// Package scala implements the Scala language reader.
package scala

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewScalaReader()) }

// ScalaReader handles Scala source files (.scala).
// Scala uses `def` as the function keyword, with brace-depth tracking.
type ScalaReader struct {
	*clike.CLikeReader
}

func NewScalaReader() *ScalaReader {
	return &ScalaReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *ScalaReader) Extensions() []string    { return []string{"scala"} }
func (r *ScalaReader) LanguageNames() []string { return []string{"scala"} }

func (r *ScalaReader) Tokenize(src []byte) iter.Seq[string] {
	return r.CLikeReader.Tokenize(src)
}

// GetConditions returns Scala CCN conditions.
func (r *ScalaReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {}, "case": {},
		"&&": {}, "||": {},
	}
}

// RunTokens drives the Scala function-detection state machine.
func (r *ScalaReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newScalaMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Scala state machine ----
// Scala uses `def` as the function keyword. Function bodies may use { } blocks
// or `=` for single-expression bodies.

type scalaMachine struct {
	m     *tokenizer.Machine
	ctx   languages.Context
	brace tokenizer.BraceTracker
}

func newScalaMachine(ctx languages.Context) *tokenizer.Machine {
	s := &scalaMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *scalaMachine) stateGlobal(tok string) bool {
	switch tok {
	case "def":
		s.ctx.PushNewFunction("")
		s.m.Next(s.stateFunctionName)
	case "{":
		s.brace.OnOpen()
	case "}":
		s.brace.OnClose(s.ctx.EndOfFunction)
	}
	return false
}

func (s *scalaMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "[":
		// Generic type parameters: def foo[T](...)
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateFunctionName, func(_ string) {}), tok)
	default:
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

func (s *scalaMachine) stateExpectFunctionDec(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "[":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateExpectFunctionDec, func(_ string) {}), tok)
	default:
		s.m.Next(s.stateExpectFunctionImpl, tok)
	}
	return false
}

func (s *scalaMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
	})
}

func (s *scalaMachine) stateExpectFunctionImpl(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImpl, "{")
	case "=":
		// Single-expression or block: def foo() = { ... } or def foo() = expr
		s.m.Next(s.stateExprBody)
	case ":":
		// Return type annotation, skip until = or {
	default:
		// Something else; stay looking
	}
	return false
}

func (s *scalaMachine) stateExprBody(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImpl, "{")
	} else {
		// Single-expression body, no braces: just confirm and return to global
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *scalaMachine) stateEnteringImpl(_ string) bool {
	s.brace.EnterFunction()
	s.m.Next(s.stateGlobal)
	return false
}
