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
	return tokenizer.GenerateTokens(src, `|--[^\n]*`) // single-line SQL comments
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
	conds := map[string]struct{}{
		"if": {}, "elsif": {}, "when": {}, "while": {}, "for": {},
		"IF": {}, "ELSIF": {}, "WHEN": {}, "WHILE": {}, "FOR": {},
		"and": {}, "or": {}, "AND": {}, "OR": {},
	}
	return conds
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
	depth int // BEGIN/END nesting depth
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
	case "PROCEDURE":
		s.m.Next(s.stateFunctionName)
	case "FUNCTION":
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func (s *plsqlMachine) stateFunctionName(tok string) bool {
	if isSpace(tok) || tok == "\n" {
		return false
	}
	s.ctx.TryNewFunction(tok)
	s.m.Next(s.stateAfterName)
	return false
}

func (s *plsqlMachine) stateAfterName(tok string) bool {
	upper := strings.ToUpper(tok)
	switch {
	case tok == "(":
		s.m.Next(s.stateParams)
	case upper == "IS" || upper == "AS":
		s.ctx.ConfirmNewFunction()
		s.m.Next(s.stateDeclSection)
	case upper == "RETURN":
		s.m.Next(s.stateReturnType)
	}
	return false
}

func (s *plsqlMachine) stateReturnType(tok string) bool {
	upper := strings.ToUpper(tok)
	if upper == "IS" || upper == "AS" {
		s.ctx.ConfirmNewFunction()
		s.m.Next(s.stateDeclSection)
	}
	return false
}

func (s *plsqlMachine) stateParams(tok string) bool {
	switch tok {
	case ")":
		s.m.Next(s.stateAfterName)
	case ",":
		// separator
	default:
		if !isSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
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
		// Nested declaration
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func (s *plsqlMachine) stateBody(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "BEGIN":
		s.depth++
	case "END":
		s.depth--
		if s.depth == 0 {
			s.ctx.EndOfFunction()
			s.m.Next(s.stateGlobal)
		}
	case "PROCEDURE", "FUNCTION":
		// Nested function in body
		s.m.Next(s.stateFunctionName)
	}
	return false
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
