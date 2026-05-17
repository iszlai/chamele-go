// Package perl implements the Perl language reader.
package perl

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewPerlReader()) }

// PerlReader handles Perl source files (.pl, .pm).
type PerlReader struct{}

func NewPerlReader() *PerlReader { return &PerlReader{} }

func (r *PerlReader) Extensions() []string    { return []string{"pl", "pm"} }
func (r *PerlReader) LanguageNames() []string { return []string{"perl"} }

func (r *PerlReader) Tokenize(src []byte) iter.Seq[string] {
	untilEnd := `(?:\\\n|[^\n])*`
	// Add single-line # comment and heredoc-ish patterns.
	return tokenizer.GenerateTokens(src, `|#`+untilEnd)
}

func (r *PerlReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "#") {
		return strings.TrimLeft(tok[1:], " \t"), true
	}
	return "", false
}

func (r *PerlReader) GetConditions() map[string]struct{} {
	// Mirror Python PerlReader._control_flow_keywords + _logical_operators + _ternary_operators.
	// ':' counts as the ternary else (Python _ternary_operators = {'?', ':'}).
	return map[string]struct{}{
		"if": {}, "elsif": {}, "unless": {}, "while": {}, "until": {},
		"for": {}, "foreach": {}, "when": {}, "given": {}, "default": {},
		"do": {}, "&&": {}, "||": {}, "?": {}, ":": {},
	}
}

// Preprocess strips whitespace and accumulates #-comments into single tokens.
func (r *PerlReader) Preprocess(tokens iter.Seq[string], _ languages.Context) iter.Seq[string] {
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

func (r *PerlReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newPerlMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Perl state machine ----
//
// Key patterns:
//   sub name { }              → named function "name"
//   sub { }                   → anonymous function "<anonymous>"
//   my $name = sub { }        → anonymous with name "$name"
//   package Foo; sub bar { }  → qualified "Foo::bar"
//
// Perl uses brace-depth tracking (like Go/Rust) since sub bodies are
// delimited by { }.

type perlMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	braceDepth  int
	funcDepths  []int
	packageName string
	pendingName string // variable name from `my $name = sub`
}

func newPerlMachine(ctx languages.Context) *tokenizer.Machine {
	s := &perlMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *perlMachine) stateGlobal(tok string) bool {
	switch tok {
	case "package":
		s.m.Next(s.statePackageName)
	case "sub":
		s.m.Next(s.stateSubName)
	case "my", "our", "local":
		s.m.Next(s.stateVarDecl)
	case "{":
		s.braceDepth++
	case "}":
		s.braceDepth--
		if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
			s.ctx.EndOfFunction()
			s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
		}
	default:
		s.pendingName = "" // reset variable name on unrelated tokens
	}
	return false
}

func (s *perlMachine) statePackageName(tok string) bool {
	if !isSpace(tok) && tok != "\n" {
		s.packageName = tok
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *perlMachine) stateVarDecl(tok string) bool {
	switch tok {
	case "$":
		// ignore sigil
	case "=", ";", "\n":
		s.m.Next(s.stateGlobal)
	default:
		if len(tok) > 0 && (isAlpha(tok[0]) || tok[0] == '_') {
			s.pendingName = "$" + tok
		}
		s.m.Next(s.stateAfterVar)
	}
	return false
}

func (s *perlMachine) stateAfterVar(tok string) bool {
	switch tok {
	case "=":
		s.m.Next(s.stateGlobal)
	case ";", "\n":
		s.pendingName = ""
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *perlMachine) stateSubName(tok string) bool {
	switch tok {
	case "{":
		// Anonymous sub: `sub {`
		name := s.qualifiedName("<anonymous>")
		if s.pendingName != "" {
			name = s.qualifiedName(s.pendingName)
		}
		s.pendingName = ""
		s.confirmFunction(name)
		return false
	case "(":
		// Prototype: `sub name($$) {`
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateSubName, func(_ string) {}), tok)
	case ";":
		// Forward declaration — skip without creating a function.
		s.m.Next(s.stateGlobal)
	default:
		if !isSpace(tok) && tok != "\n" && len(tok) > 0 && (isAlpha(tok[0]) || tok[0] == '_') {
			// Named sub.
			name := s.qualifiedName(tok)
			s.pendingName = ""
			s.m.Next(s.stateSubBody(name))
		}
	}
	return false
}

// stateSubBody waits for `{` or `;` after sub declaration.
func (s *perlMachine) stateSubBody(name string) tokenizer.StateFn {
	return func(tok string) bool {
		switch tok {
		case "{":
			s.confirmFunction(name)
		case ";":
			// Forward declaration — no body.
			s.m.Next(s.stateGlobal)
		}
		return false
	}
}

func (s *perlMachine) confirmFunction(name string) {
	s.ctx.PushNewFunction(name)
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
}

func (s *perlMachine) qualifiedName(name string) string {
	if s.packageName != "" {
		return s.packageName + "::" + name
	}
	return name
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}

func isHSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
