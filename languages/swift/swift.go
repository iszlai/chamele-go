// Package swift implements the Swift language reader.
package swift

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewSwiftReader()) }

// SwiftReader handles Swift source files (.swift).
// Swift uses `func` as the function keyword, also recognising `init`, `get`,
// `set`, `willSet`, `didSet`, and `deinit` as function-like constructs.
type SwiftReader struct {
	*clike.CLikeReader
}

func NewSwiftReader() *SwiftReader {
	return &SwiftReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *SwiftReader) Extensions() []string    { return []string{"swift"} }
func (r *SwiftReader) LanguageNames() []string { return []string{"swift"} }

func (r *SwiftReader) Tokenize(src []byte) iter.Seq[string] {
	// Add Swift-specific: backtick identifiers, optional types (Foo?), forced unwrap (Foo!), ??.
	return tokenizer.GenerateTokens(src, clike.CLikeAddition()+
		"`"+`\w+`+"`"+
		`|\w+\?`+
		`|\w+!`+
		`|\?\?`)
}

// GetConditions returns Swift CCN conditions.
func (r *SwiftReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {}, "guard": {}, "case": {},
		"&&": {}, "||": {}, "?": {},
	}
}

// RunTokens drives the Swift function-detection state machine.
func (r *SwiftReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newSwiftMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Swift state machine ----

type swiftMachine struct {
	m          *tokenizer.Machine
	ctx        languages.Context
	braceDepth int
	funcDepths []int
}

func newSwiftMachine(ctx languages.Context) *tokenizer.Machine {
	s := &swiftMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *swiftMachine) stateGlobal(tok string) bool {
	switch tok {
	case "func":
		s.ctx.PushNewFunction("")
		s.m.Next(s.stateFunctionName)
	case "init", "subscript":
		// Treat as named functions
		s.ctx.PushNewFunction(tok)
		s.m.Next(s.stateExpectFunctionDec)
	case "get", "set", "willSet", "didSet", "deinit":
		// Accessor bodies count as functions
		s.ctx.PushNewFunction(tok)
		s.m.Next(s.stateExpectFunctionImpl)
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

func (s *swiftMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateFunctionName, func(_ string) {}), tok)
	default:
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

func (s *swiftMachine) stateExpectFunctionDec(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateExpectFunctionDec, func(_ string) {}), tok)
	default:
		s.m.Next(s.stateExpectFunctionImpl, tok)
	}
	return false
}

func (s *swiftMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
	})
}

func (s *swiftMachine) stateExpectFunctionImpl(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImpl, "{")
	}
	return false
}

func (s *swiftMachine) stateEnteringImpl(_ string) bool {
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}
