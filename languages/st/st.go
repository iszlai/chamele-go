// Package st implements the Structured Text (IEC 61131-3) language reader.
package st

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewSTReader()) }

// STReader handles Structured Text source files (.st).
// ST uses FUNCTION_BLOCK/FUNCTION/ACTION name ... END_FUNCTION_BLOCK/END_FUNCTION/END_ACTION.
type STReader struct{}

func NewSTReader() *STReader { return &STReader{} }

func (r *STReader) Extensions() []string    { return []string{"st"} }
func (r *STReader) LanguageNames() []string { return []string{"st"} }

func (r *STReader) Tokenize(src []byte) iter.Seq[string] {
	untilEnd := `(?:\\\n|[^\n])*`
	return tokenizer.GenerateTokens(src,
		`|//`+untilEnd+ // line comments
			`|\(\*`+untilEnd+ // block comments (* ... *)
			`|FUNCTION_BLOCK`+
			`|END_FUNCTION_BLOCK`+
			`|END_FUNCTION`+
			`|END_ACTION`)
}

func (r *STReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "(*") || strings.HasPrefix(tok, "//") {
		return tok[2:], true
	}
	return "", false
}

func (r *STReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elsif": {}, "for": {}, "while": {}, "repeat": {}, "case": {},
		"IF": {}, "ELSIF": {}, "FOR": {}, "WHILE": {}, "REPEAT": {}, "CASE": {},
		"and": {}, "or": {}, "AND": {}, "OR": {},
	}
}

// Preprocess strips whitespace, keeps newlines, and normalises END_* tokens.
func (r *STReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "\n" || !isHSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}

func (r *STReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newSTMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- ST state machine ----

type stMachine struct {
	m      *tokenizer.Machine
	ctx    languages.Context
	depth  int
	inFunc bool
}

func newSTMachine(ctx languages.Context) *tokenizer.Machine {
	s := &stMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *stMachine) stateGlobal(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "FUNCTION_BLOCK", "FUNCTION", "ACTION":
		s.m.Next(s.stateFunctionName)
	case "IF", "FOR", "WHILE", "CASE", "REPEAT":
		if s.inFunc {
			s.depth++
		}
	case "END_IF", "END_FOR", "END_WHILE", "END_CASE", "END_REPEAT":
		if s.inFunc && s.depth > 0 {
			s.depth--
		}
	case "END_FUNCTION_BLOCK", "END_FUNCTION", "END_ACTION":
		if s.inFunc {
			s.ctx.EndOfFunction()
			s.inFunc = false
			s.depth = 0
		}
	}
	return false
}

func (s *stMachine) stateFunctionName(tok string) bool {
	if isHSpace(tok) || tok == "\n" {
		return false
	}
	s.ctx.RestartNewFunction(tok)
	s.inFunc = true
	s.depth = 0
	s.m.Next(s.stateGlobal)
	return false
}

func isHSpace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' {
			return false
		}
	}
	return len(s) > 0
}
