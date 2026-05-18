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
	// Compound tokens must come before \w+ to take precedence.
	// [^\S\n]+ = horizontal whitespace (spaces/tabs, no newlines).
	return tokenizer.GenerateTokens(src,
		`|!`+untilEnd+ // Fortran ! comments
			`|(?i)end[^\S\n]+(?:if|do|select|selectcase|subroutine|function|module|program)`+ // END <kw>
			`|(?i)else[^\S\n]+if`+ // ELSE IF compound (not a new block)
			`|\.AND\.`+
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
		"IF": {}, "if": {},
		"DO": {}, "do": {},
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
// Block IF: IF ... THEN ... END IF (ELSE IF doesn't open a new depth level).
// Compound tokenizer patterns prevent double-counting keywords in END IF, END DO, ELSE IF.

type fortranMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	depth       int  // nesting depth of THEN/DO/SELECT blocks
	afterElseIf bool // compound ELSE IF seen — next THEN should not open a new block
	inModule    bool
}

func newFortranMachine(ctx languages.Context) *tokenizer.Machine {
	s := &fortranMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

// normalizeToken normalizes compound tokens like "end  if" → "END IF".
func normalizeToken(tok string) string {
	fields := strings.Fields(strings.ToUpper(tok))
	return strings.Join(fields, " ")
}

func (s *fortranMachine) stateGlobal(tok string) bool {
	upper := normalizeToken(tok)

	switch upper {
	case "SUBROUTINE", "FUNCTION":
		s.afterElseIf = false
		s.m.Next(s.stateFunctionName)
	case "PROGRAM":
		s.afterElseIf = false
		s.m.Next(s.stateSkipName)
	case "MODULE":
		s.afterElseIf = false
		s.m.Next(s.stateModuleName)
	case "ELSE IF":
		// Compound: the next THEN is part of this ELSE IF clause, not a new block.
		s.afterElseIf = true
	case "IF":
		// Standalone IF: next THEN opens a new block.
		s.afterElseIf = false
	case "THEN":
		if !s.afterElseIf {
			s.depth++
		}
		s.afterElseIf = false
	case "DO":
		s.afterElseIf = false
		s.depth++
	case "SELECT":
		s.afterElseIf = false
		s.depth++
	case "END":
		s.afterElseIf = false
		s.m.Next(s.stateAfterEnd)
	case "END IF", "ENDIF":
		s.afterElseIf = false
		if s.depth > 0 {
			s.depth--
		}
	case "END DO", "ENDDO":
		s.afterElseIf = false
		if s.depth > 0 {
			s.depth--
		}
	case "END SELECT", "ENDSELECT", "END SELECTCASE", "ENDSELECTCASE":
		s.afterElseIf = false
		if s.depth > 0 {
			s.depth--
		}
	case "END SUBROUTINE", "ENDSUBROUTINE":
		s.afterElseIf = false
		if s.depth > 0 {
			s.depth--
		} else {
			s.ctx.EndOfFunction()
		}
	case "END FUNCTION", "ENDFUNCTION":
		s.afterElseIf = false
		if s.depth > 0 {
			s.depth--
		} else {
			s.ctx.EndOfFunction()
		}
	case "END MODULE", "ENDMODULE", "END PROGRAM", "ENDPROGRAM":
		s.afterElseIf = false
		if s.inModule {
			s.ctx.PopNesting()
			s.inModule = false
		}
		// default: don't change afterElseIf (persists through condition expressions)
	}
	return false
}

// stateAfterEnd handles the token following a bare END keyword (for unusual spacing
// not caught by the compound tokenizer patterns).
func (s *fortranMachine) stateAfterEnd(tok string) bool {
	upper := normalizeToken(tok)
	switch upper {
	case "IF", "DO", "SELECT", "SELECTCASE":
		if s.depth > 0 {
			s.depth--
		}
		s.m.Next(s.stateGlobal)
	case "SUBROUTINE", "FUNCTION":
		if s.depth > 0 {
			s.depth--
		} else {
			s.ctx.EndOfFunction()
		}
		s.m.Next(s.stateSkipName)
	case "MODULE", "PROGRAM":
		if s.inModule {
			s.ctx.PopNesting()
			s.inModule = false
		}
		s.m.Next(s.stateSkipName)
	default:
		// Bare END
		if s.depth > 0 {
			s.depth--
		} else {
			s.ctx.EndOfFunction()
		}
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *fortranMachine) stateModuleName(tok string) bool {
	if tok == "\n" || isSpace(tok) {
		return false
	}
	s.ctx.AddNamespace(tok)
	s.inModule = true
	s.depth = 0
	s.m.Next(s.stateGlobal)
	return false
}

func (s *fortranMachine) stateFunctionName(tok string) bool {
	if tok == "\n" || isSpace(tok) {
		return false
	}
	s.ctx.RestartNewFunction(tok)
	s.ctx.AddToLongFunctionName("(")
	s.depth = 0
	s.afterElseIf = false
	s.m.Next(s.stateAfterName)
	return false
}

func (s *fortranMachine) stateAfterName(tok string) bool {
	switch tok {
	case "(":
		s.m.Next(s.stateParams)
	default:
		// Start function body — re-process in stateGlobal.
		s.m.Next(s.stateGlobal, tok)
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
	upper := normalizeToken(tok)
	switch upper {
	case "SUBROUTINE", "FUNCTION", "PROGRAM", "MODULE", "END":
		s.m.Next(s.stateGlobal, tok)
	default:
		s.m.Next(s.stateGlobal)
	}
	return false
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
