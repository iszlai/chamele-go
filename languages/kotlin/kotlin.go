// Package kotlin implements the Kotlin language reader.
package kotlin

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewKotlinReader()) }

// KotlinReader handles Kotlin source files (.kt, .kts).
// Kotlin uses `fun` as the function keyword, with Go-like brace-depth tracking.
type KotlinReader struct {
	*clike.CLikeReader
}

func NewKotlinReader() *KotlinReader {
	return &KotlinReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *KotlinReader) Extensions() []string    { return []string{"kt", "kts"} }
func (r *KotlinReader) LanguageNames() []string { return []string{"kotlin"} }

func (r *KotlinReader) Tokenize(src []byte) iter.Seq[string] {
	// Add backtick identifiers, nullable types (Foo?), non-null asserts (Foo!!),
	// and Kotlin's null-coalescing (??) and elvis (?:) operators.
	return tokenizer.GenerateTokens(src, clike.CLikeAddition()+
		"`"+`\w+`+"`"+
		`|\w+\?`+
		`|\w+!!`+
		`|\?\?`+
		`|\?:`)
}

// GetConditions returns Kotlin CCN conditions.
func (r *KotlinReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {},
		"&&": {}, "||": {}, "?:": {},
	}
}

// RunTokens drives the Kotlin function-detection state machine.
func (r *KotlinReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newKotlinMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Kotlin state machine ----
// Kotlin uses `fun` as the function keyword and tracks brace depth explicitly,
// identical to the Go reader approach. Methods can appear in class bodies;
// CLikeNestingStackStates is intentionally not used to avoid interaction.

type kotlinMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	braceDepth int
	funcDepths []int
}

func newKotlinMachine(ctx languages.Context) *tokenizer.Machine {
	s := &kotlinMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *kotlinMachine) stateGlobal(tok string) bool {
	switch tok {
	case "fun":
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

func (s *kotlinMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		// Generic type parameters: fun <T> name(...)
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateFunctionName, func(_ string) {}), tok)
	case "{":
		// Anonymous or single-expression function: fun { ... }
		s.m.Next(s.stateEnteringImpl, "{")
	default:
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

func (s *kotlinMachine) stateExpectFunctionDec(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateExpectFunctionDec, func(_ string) {}), tok)
	default:
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *kotlinMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
	})
}

func (s *kotlinMachine) stateExpectFunctionImpl(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImpl, "{")
	case "=":
		// Single-expression function: fun foo() = expr
		// Count it but no braces; end at next statement boundary.
		s.ctx.PushNewFunction(s.ctx.CurrentFunctionName())
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *kotlinMachine) stateEnteringImpl(_ string) bool {
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}
