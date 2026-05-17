// Package r implements the R language reader.
package r

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewRReader()) }

// RReader handles R source files (.r, .R).
// R uses `name <- function(params) { body }` or `name = function(params) { body }`.
type RReader struct{}

func NewRReader() *RReader { return &RReader{} }

func (r *RReader) Extensions() []string    { return []string{"r", "R"} }
func (r *RReader) LanguageNames() []string { return []string{"r", "R"} }

func (r *RReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src,
		`|<-`+     // assignment operator
			`|->`+ // right assignment
			`|%[a-zA-Z_*/>]+%`+ // special operators %in%, %*%
			`|\.\.\.|`+ // ellipsis
			`|:::|`+    // internal namespace
			`|::`)      // namespace
}

func (r *RReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "#") {
		return strings.TrimSpace(tok[1:]), true
	}
	return "", false
}

func (r *RReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "repeat": {},
		"&&": {}, "||": {}, "&": {}, "|": {},
	}
}

func (r *RReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newRMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
	m.StatemachineBeforeReturn()
}

// ---- R state machine ----
// R function pattern: name <- function(params) { body }
// We track the most recent identifier before <- or = to use as function name.

type rMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	lastIdent   string // last seen identifier (potential function name)
	braceDepth  int
	funcDepths  []int
}

func newRMachine(ctx languages.Context) *tokenizer.Machine {
	s := &rMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *rMachine) stateGlobal(tok string) bool {
	switch tok {
	case "function":
		// Confirmed function keyword; use lastIdent as name
		name := s.lastIdent
		if name == "" {
			name = "(anonymous)"
		}
		s.lastIdent = ""
		s.ctx.RestartNewFunction(name)
		s.m.Next(s.stateExpectParams)
	case "<-", "=":
		// Assignment: name <- function(...)
		// Keep lastIdent as the potential name
	case "{":
		s.braceDepth++
	case "}":
		s.braceDepth--
		if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
			s.ctx.EndOfFunction()
			s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
		}
	default:
		if !isSpace(tok) && tok != "\n" && tok != "#" {
			// Track the last identifier seen
			if len(tok) > 0 && (isAlpha(tok[0]) || tok[0] == '_' || tok[0] == '.') {
				s.lastIdent = tok
			} else if tok != "<-" && tok != "=" {
				s.lastIdent = ""
			}
		}
	}
	return false
}

func (s *rMachine) stateExpectParams(tok string) bool {
	if tok == "(" {
		s.ctx.AddToLongFunctionName("(")
		s.m.Next(s.stateParams)
	}
	return false
}

func (s *rMachine) stateParams(tok string) bool {
	switch tok {
	case ")":
		s.ctx.AddToLongFunctionName(")")
		s.m.Next(s.stateExpectBody)
	case ",":
		// separator
	default:
		if !isSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
			s.ctx.AddToLongFunctionName(" " + tok)
		}
	}
	return false
}

func (s *rMachine) stateExpectBody(tok string) bool {
	switch tok {
	case "{":
		s.funcDepths = append(s.funcDepths, s.braceDepth)
		s.braceDepth++
		s.m.Next(s.stateGlobal)
	case "\n":
		// Single-line function body: name <- function() expr\n
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *rMachine) StatemachineBeforeReturn() {
	// End any open single-line function
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
