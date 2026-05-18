// Package golang implements the Go language reader.
// (Package is named "golang" to avoid conflict with the "go" keyword.)
package golang

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() {
	languages.Register(NewGoReader())
}

// GoReader is the language reader for Go source files.
type GoReader struct{}

func NewGoReader() *GoReader { return &GoReader{} }

func (r *GoReader) Extensions() []string    { return []string{"go"} }
func (r *GoReader) LanguageNames() []string { return []string{"go"} }

func (r *GoReader) Tokenize(src []byte) iter.Seq[string] {
	// Add backtick raw-string literals to the base token pattern.
	return tokenizer.GenerateTokens(src, "`[^`]*`")
}

func (r *GoReader) GetComment(tok string) (string, bool) {
	if len(tok) >= 2 && (tok[:2] == "//" || tok[:2] == "/*") {
		return tok[2:], true
	}
	return "", false
}

func (r *GoReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "case": {}, "&&": {}, "||": {},
	}
}

func (r *GoReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newGoMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Go state machine ----
//
// Design: brace depth is tracked explicitly rather than using sub_state clones.
// funcDepths records the depth (before the `{`) of each confirmed function body,
// so the matching `}` triggers EndOfFunction().

type goMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	braceDepth int
	funcDepths []int // depth before each function's `{`; popped on matching `}`
	lastToken  string
}

func newGoMachine(ctx languages.Context) *tokenizer.Machine {
	s := &goMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *goMachine) stateGlobal(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "func":
		s.ctx.PushNewFunction("")
		s.m.Next(s.stateFunctionName)
	case "type":
		s.m.Next(s.stateTypeName)
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

// stateFunctionName: reads function name, handles receiver `(receiver)`.
//
// Branch logic mirrors golike.py:
//   - global scope (IsInsideFunction=false): `(` → stateMemberFunction (ends at stateFunctionName)
//   - inside a function (IsInsideFunction=true): `(` → stateFunctionDec (ends at stateExpectFunctionImpl)
//
// At global scope, the first `(` after `func` is the receiver list;
// stateMemberFunction reads it into long_name and returns to stateFunctionName.
// Inside a function, the `(` starts a closure's parameter list.
func (s *goMachine) stateFunctionName(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "`" {
		return false
	}
	switch tok {
	case "(":
		if s.ctx.IsInsideFunction() {
			// Inside a real function: treat ( as closure parameter list.
			s.m.Next(s.stateFunctionDec(), tok)
		} else {
			// At global scope: treat ( as receiver list (adds to long_name).
			s.m.Next(s.stateMemberFunction(), tok)
		}
	case "{":
		s.m.Next(s.stateExpectFunctionImpl, "{")
	default:
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

// stateExpectFunctionDec: after reading the function name; wait for `(`.
func (s *goMachine) stateExpectFunctionDec(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		// Go generics: `func Name[T any]` — skip type parameters
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateExpectFunctionDec, func(_ string) {}))
	case "[":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateExpectFunctionDec, func(_ string) {}), tok)
	default:
		s.m.Next(s.stateGlobal)
	}
	return false
}

// stateMemberFunction: reads receiver `(RecvType)` then continues to function name.
func (s *goMachine) stateMemberFunction() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateFunctionName, func(tok string) {
		s.ctx.AddToLongFunctionName(tok)
	})
}

// stateFunctionDec: reads parameter list `(params)`.
func (s *goMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
	})
}

// stateExpectFunctionImpl: after parameter list; wait for `{` (skip return types).
func (s *goMachine) stateExpectFunctionImpl(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "{" && s.lastToken != "interface" {
		s.m.Next(s.stateEnteringImpl, "{")
	}
	return false
}

// stateEnteringImpl: `{` confirms the function body.
func (s *goMachine) stateEnteringImpl(tok string) bool {
	defer func() { s.lastToken = tok }()
	// Record current depth BEFORE incrementing (so `}` at that depth closes this function).
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}

// stateTypeName: skip the type name token, then stateAfterTypeName.
func (s *goMachine) stateTypeName(tok string) bool {
	s.m.Next(s.stateAfterTypeName)
	return false
}

// stateAfterTypeName: `struct` and `interface` bodies are skipped entirely.
func (s *goMachine) stateAfterTypeName(tok string) bool {
	switch tok {
	case "struct", "interface":
		s.m.Next(s.stateSkipBraceBlock())
	default:
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *goMachine) stateSkipBraceBlock() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "{", "}", s.stateGlobal, func(_ string) {})
}

// Preprocess strips whitespace but keeps newlines.
func (r *GoReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "\n" || !stringx.IsHSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}
