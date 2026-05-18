// Package rust implements the Rust language reader.
package rust

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewRustReader()) }

// RustReader handles Rust source files (.rs).
type RustReader struct{}

func NewRustReader() *RustReader { return &RustReader{} }

func (r *RustReader) Extensions() []string    { return []string{"rs"} }
func (r *RustReader) LanguageNames() []string { return []string{"rust"} }

func (r *RustReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, `|(?:'\w+\b)`) // lifetimes / labels
}

func (r *RustReader) GetComment(tok string) (string, bool) {
	if len(tok) >= 2 && (tok[:2] == "//" || tok[:2] == "/*") {
		return tok[2:], true
	}
	return "", false
}

func (r *RustReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "match": {}, "&&": {}, "||": {},
	}
}

func (r *RustReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newRustMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Rust state machine ----
//
// PushNewFunction is intentionally deferred to stateEnteringImpl (when `{` is
// confirmed). This means trait/interface method declarations that end with `;`
// simply discard the accumulated candidate — no cleanup needed.

type rustMachine struct {
	m             *tokenizer.Machine
	ctx           languages.Context
	brace         tokenizer.BraceTracker
	pendingName   string
	pendingParams []string
	bracketStack  []string
}

func newRustMachine(ctx languages.Context) *tokenizer.Machine {
	s := &rustMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *rustMachine) stateGlobal(tok string) bool {
	switch tok {
	case "fn":
		s.pendingName = ""
		s.pendingParams = nil
		s.m.Next(s.stateFunctionName)
	case "{":
		s.brace.OnOpen()
	case "}":
		s.brace.OnClose(s.ctx.EndOfFunction)
	}
	return false
}

func (s *rustMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.bracketStack = nil
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateFunctionName, func(_ string) {}), tok)
	case "{":
		// `fn {` is invalid Rust but handle gracefully by entering body with pending name.
		s.m.Next(s.stateEnteringImpl, tok)
	case ";", "}":
		// Discard candidate; pass token back to global.
		s.m.Next(s.stateGlobal, tok)
	default:
		s.pendingName = tok
		s.m.Next(s.stateExpectFunctionDec)
	}
	return false
}

func (s *rustMachine) stateExpectFunctionDec(tok string) bool {
	switch tok {
	case "(":
		s.bracketStack = nil
		s.m.Next(s.stateFunctionDec(), tok)
	case "<":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateExpectFunctionDec, func(_ string) {}), tok)
	case ";", "}":
		// No body → discard candidate and return token to global.
		s.m.Next(s.stateGlobal, tok)
	default:
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *rustMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectFunctionImpl, func(tok string) {
		switch {
		case tok == "(" || tok == "<":
			s.bracketStack = append(s.bracketStack, tok)
		case tok == ")" || tok == ">":
			if len(s.bracketStack) > 0 {
				s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			}
		case len(s.bracketStack) == 1:
			if tok != "void" && tok != "," {
				s.pendingParams = append(s.pendingParams, tok)
			}
		}
	})
}

func (s *rustMachine) stateExpectFunctionImpl(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImpl, tok)
	case ";":
		// Trait/interface method declaration — no body, discard candidate.
		s.m.Next(s.stateGlobal)
	case "}":
		// Block closed before seeing body — discard and return to global.
		s.m.Next(s.stateGlobal, tok)
	}
	// Return types, where clauses, etc. are silently skipped.
	return false
}

func (s *rustMachine) stateEnteringImpl(_ string) bool {
	// Only NOW commit the function — avoids "ghost" entries for trait declarations.
	name := s.pendingName
	s.ctx.PushNewFunction(name)
	for _, p := range s.pendingParams {
		s.ctx.Parameter(p)
	}
	s.pendingParams = nil
	s.pendingName = ""
	s.brace.EnterFunction()
	s.m.Next(s.stateGlobal)
	return false
}
