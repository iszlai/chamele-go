// Package ruby implements the Ruby language reader.
package ruby

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewRubyReader()) }

// RubyReader handles Ruby source files (.rb).
// Ruby uses `def name ... end` to define functions.
type RubyReader struct{}

func NewRubyReader() *RubyReader { return &RubyReader{} }

func (r *RubyReader) Extensions() []string    { return []string{"rb"} }
func (r *RubyReader) LanguageNames() []string { return []string{"ruby"} }

func (r *RubyReader) Tokenize(src []byte) iter.Seq[string] {
	untilEnd := `(?:\\\n|[^\n])*`
	return tokenizer.GenerateTokens(src,
		`|#`+untilEnd+
			`|\w+[?!]`+ // method names like foo? or bar!
			`|:\w+`+ // symbols
			`|\$\w+`+ // global variables
			`|@{0,2}\w+`) // instance/class variables
}

func (r *RubyReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "#") {
		return strings.TrimSpace(tok[1:]), true
	}
	return "", false
}

func (r *RubyReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elsif": {}, "unless": {}, "for": {}, "while": {}, "until": {},
		"rescue": {}, "when": {},
		"&&": {}, "||": {}, "and": {}, "or": {}, "?": {},
	}
}

func (r *RubyReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newRubyMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Ruby state machine ----
// Ruby uses def ... end blocks. We track nesting depth with do/begin/class/module/{
// incrementing and end/} decrementing. When depth returns to 0 after a def, the
// function ends.

type rubyMachine struct {
	m         *tokenizer.Machine
	ctx       languages.Context
	depth     int  // nesting depth within current def block
	inFunc    bool // are we inside a function body?
	lastToken string
}

func newRubyMachine(ctx languages.Context) *tokenizer.Machine {
	s := &rubyMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *rubyMachine) stateGlobal(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "def":
		s.m.Next(s.stateFunctionName)
	case "do", "begin", "class", "module", "{":
		if s.lastToken != "." {
			if s.inFunc {
				s.depth++
			}
		}
	case "end", "}":
		if s.inFunc {
			if s.depth > 0 {
				s.depth--
			} else {
				s.ctx.EndOfFunction()
				s.inFunc = false
			}
		}
	}
	return false
}

func (s *rubyMachine) stateFunctionName(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		// Anonymous function: def (params)
		s.ctx.PushNewFunction("(anonymous)")
		s.m.Next(s.stateFunctionParams)
	case "\n", ";":
		// def without explicit name — ignore
		s.m.Next(s.stateGlobal)
	default:
		s.ctx.PushNewFunction(tok)
		s.m.Next(s.stateAfterName)
	}
	return false
}

func (s *rubyMachine) stateAfterName(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionParams)
	case "\n", ";":
		s.inFunc = true
		s.depth = 0
		s.m.Next(s.stateGlobal)
	case ".":
		// Class method: def Foo.bar
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateMethodSuffix)
	default:
		// Some modifier (e.g. `def self.foo`)
		s.ctx.AddToFunctionName(" " + tok)
	}
	return false
}

func (s *rubyMachine) stateMethodSuffix(tok string) bool {
	s.ctx.AddToFunctionName(tok)
	s.m.Next(s.stateAfterName)
	return false
}

func (s *rubyMachine) stateFunctionParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case ")":
		s.inFunc = true
		s.depth = 0
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

func isSpace(s string) bool {
	return strings.TrimLeft(s, " \t\r") == "" && len(s) > 0
}
