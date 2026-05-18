// Package plsql implements the PL/SQL language reader.
package plsql

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewPLSQLReader()) }

// PLSQLReader handles PL/SQL source files (.sql, .pls, .pkb, .pks).
// PL/SQL uses PROCEDURE/FUNCTION name(...) IS/AS ... BEGIN ... END; syntax.
type PLSQLReader struct{}

func NewPLSQLReader() *PLSQLReader { return &PLSQLReader{} }

func (r *PLSQLReader) Extensions() []string {
	return []string{"sql", "pls", "pkb", "pks", "plb", "pck"}
}
func (r *PLSQLReader) LanguageNames() []string { return []string{"plsql", "pl/sql"} }

func (r *PLSQLReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src,
		`|--[^\n]*`+ // single-line SQL comments
			`|(?i)end\s+if`+ // "END IF" as compound token (prevents IF being double-counted)
			`|(?i)end\s+loop`+ // "END LOOP"
			`|(?i)end\s+case`) // "END CASE"
}

func (r *PLSQLReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "--") {
		return tok[2:], true
	}
	if len(tok) >= 2 && tok[:2] == "/*" {
		return tok[2:], true
	}
	return "", false
}

func (r *PLSQLReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elsif": {}, "when": {}, "while": {}, "for": {},
		"IF": {}, "ELSIF": {}, "WHEN": {}, "WHILE": {}, "FOR": {},
		"and": {}, "or": {}, "AND": {}, "OR": {},
	}
}

func (r *PLSQLReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newPLSQLMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- PL/SQL state machine ----

type plsqlMachine struct {
	m     *tokenizer.Machine
	ctx   languages.Context
	depth int // BEGIN/END nesting depth (only BEGIN increments)
}

func newPLSQLMachine(ctx languages.Context) *tokenizer.Machine {
	s := &plsqlMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *plsqlMachine) stateGlobal(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "PROCEDURE", "FUNCTION":
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func (s *plsqlMachine) stateFunctionName(tok string) bool {
	if isSpace(tok) || tok == "\n" {
		return false
	}
	// Use PushNewFunction so nested procedures stack correctly.
	s.ctx.PushNewFunction(tok)
	s.m.Next(s.stateAfterName)
	return false
}

func (s *plsqlMachine) stateAfterName(tok string) bool {
	upper := strings.ToUpper(tok)
	switch {
	case tok == "(":
		s.m.Next(s.stateParams)
	case upper == "IS" || upper == "AS":
		s.m.Next(s.stateDeclSection)
	case upper == "RETURN":
		s.m.Next(s.stateReturnType)
	}
	return false
}

func (s *plsqlMachine) stateReturnType(tok string) bool {
	upper := strings.ToUpper(tok)
	if upper == "IS" || upper == "AS" {
		s.m.Next(s.stateDeclSection)
	}
	return false
}

func (s *plsqlMachine) stateParams(tok string) bool {
	switch tok {
	case ")":
		s.m.Next(s.stateAfterName)
	default:
		if !isSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok) // "," calls AddParameter(",") which starts a new slot
		}
	}
	return false
}

func (s *plsqlMachine) stateDeclSection(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "BEGIN":
		s.depth = 1
		s.m.Next(s.stateBody)
	case "PROCEDURE", "FUNCTION":
		// Nested declaration — recurse
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func (s *plsqlMachine) stateBody(tok string) bool {
	// Normalize compound tokens like "end  if" → "END IF"
	fields := strings.Fields(strings.ToUpper(tok))
	upper := strings.Join(fields, " ")

	switch upper {
	case "BEGIN":
		s.depth++
	case "END IF", "END LOOP", "END CASE":
		// Block closers — do NOT affect BEGIN/END depth
	case "END":
		s.m.Next(s.stateAfterEnd)
	case "PROCEDURE", "FUNCTION":
		// Nested function defined in body
		s.m.Next(s.stateFunctionName)
	}
	return false
}

// stateAfterEnd handles the token following a bare END keyword.
// Compound tokens "END IF", "END LOOP", "END CASE" are produced by the tokenizer
// and handled in stateBody directly; stateAfterEnd handles remaining bare END cases.
func (s *plsqlMachine) stateAfterEnd(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "LOOP", "IF", "CASE":
		// Fallback for non-compound forms (unusual spacing)
		s.m.Next(s.stateBody)
	default:
		// "END;" or "END name;" — closes a BEGIN block
		nested := s.ctx.IsInsideFunction()
		s.depth--
		if s.depth == 0 {
			s.ctx.EndOfFunction()
			if nested {
				// Return to outer function's declaration section
				s.m.Next(s.stateDeclSection, tok)
			} else {
				s.m.Next(s.stateGlobal, tok)
			}
		} else {
			s.m.Next(s.stateBody, tok)
		}
	}
	return false
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
