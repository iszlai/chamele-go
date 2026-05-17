// Package fortran implements the Fortran language reader.
package fortran

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewFortranReader()) }

// FortranReader handles Fortran source files (.f90, .f95, .f03, .f08).
// Fortran is case-insensitive; SUBROUTINE and FUNCTION keywords start a function.
type FortranReader struct{}

func NewFortranReader() *FortranReader { return &FortranReader{} }

func (r *FortranReader) Extensions() []string {
	return []string{"f90", "f95", "f03", "f08", "f", "for"}
}
func (r *FortranReader) LanguageNames() []string { return []string{"fortran"} }

func (r *FortranReader) Tokenize(src []byte) iter.Seq[string] {
	untilEnd := `(?:\\\n|[^\n])*`
	return tokenizer.GenerateTokens(src,
		`|!`+untilEnd+ // Fortran ! comments
			`|\.AND\.`+ // logical operators
			`|\.OR\.`+
			`|\.and\.`+
			`|\.or\.`)
}

func (r *FortranReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "!") {
		return tok[1:], true
	}
	return "", false
}

func (r *FortranReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"IF": {}, "if": {}, "DO": {}, "do": {},
		".AND.": {}, ".OR.": {}, ".and.": {}, ".or.": {},
		"CASE": {}, "case": {},
	}
}

func (r *FortranReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newFortranMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Fortran state machine ----
// SUBROUTINE name(params) and FUNCTION name(params) introduce functions.
// END SUBROUTINE / END FUNCTION close them. Nesting is tracked by depth.

type fortranMachine struct {
	m     *tokenizer.Machine
	ctx   languages.Context
	depth int
}

func newFortranMachine(ctx languages.Context) *tokenizer.Machine {
	s := &fortranMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *fortranMachine) stateGlobal(tok string) bool {
	upper := strings.ToUpper(tok)
	switch upper {
	case "SUBROUTINE", "FUNCTION":
		s.m.Next(s.stateFunctionName)
	case "PROGRAM", "MODULE":
		s.m.Next(s.stateSkipName)
	case "IF", "DO":
		s.depth++ // IF THEN / DO blocks increment depth
	case "END":
		if s.depth > 0 {
			s.depth--
		} else {
			s.ctx.EndOfFunction()
		}
	}
	// Check for "END SUBROUTINE", "END FUNCTION", "ENDSUBROUTINE", "ENDFUNCTION"
	if strings.HasPrefix(upper, "END") && len(upper) > 3 {
		rest := upper[3:]
		if rest == "SUBROUTINE" || rest == "FUNCTION" || rest == "PROGRAM" || rest == "MODULE" {
			if s.depth > 0 {
				s.depth--
			} else {
				s.ctx.EndOfFunction()
			}
		}
	}
	return false
}

func (s *fortranMachine) stateFunctionName(tok string) bool {
	// Skip whitespace/empty tokens
	if tok == "\n" || isSpace(tok) {
		return false
	}
	s.ctx.RestartNewFunction(tok)
	s.ctx.AddToLongFunctionName("(")
	s.depth = 0
	s.m.Next(s.stateAfterName)
	return false
}

func (s *fortranMachine) stateAfterName(tok string) bool {
	if tok == "(" {
		s.m.Next(s.stateParams)
	} else if tok == "\n" {
		// No params: SUBROUTINE foo
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *fortranMachine) stateParams(tok string) bool {
	switch tok {
	case ")":
		s.ctx.AddToLongFunctionName(" )")
		s.m.Next(s.stateGlobal)
	case ",":
		// separator
	default:
		if !isSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
		}
	}
	return false
}

func (s *fortranMachine) stateSkipName(tok string) bool {
	// Skip the program/module name
	s.m.Next(s.stateGlobal)
	return false
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
