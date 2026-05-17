// Package lua implements the Lua language reader.
package lua

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() { languages.Register(NewLuaReader()) }

// LuaReader handles Lua source files (.lua).
// Lua uses `function name() ... end` or `name = function() ... end`.
type LuaReader struct{}

func NewLuaReader() *LuaReader { return &LuaReader{} }

func (r *LuaReader) Extensions() []string    { return []string{"lua"} }
func (r *LuaReader) LanguageNames() []string { return []string{"lua"} }

func (r *LuaReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src,
		`|\-\-\[\[.*?\]\]`+ // block comments --[[ ... ]]
			`|\-\-[^\n]*`) // line comments --...
}

func (r *LuaReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "--") {
		return tok[2:], true
	}
	return "", false
}

func (r *LuaReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elseif": {}, "for": {}, "while": {}, "repeat": {}, "and": {}, "or": {},
	}
}

func (r *LuaReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newLuaMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Lua state machine ----
// Lua has two styles:
//  1. function name() ... end
//  2. name = function() ... end  (assignment form)
//
// Nesting: do/then/function/repeat all open a block; end closes one.

type luaMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	depth       int    // nesting depth
	inFunc      bool   // inside a function
	funcDepth   int    // depth at function entry (for matching end)
	lastToken   string // most recent non-whitespace token for name resolution
	pendingName string // name from LHS of assignment
}

func newLuaMachine(ctx languages.Context) *tokenizer.Machine {
	s := &luaMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *luaMachine) stateGlobal(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "function":
		// Next token may be a name or "(" (anonymous)
		s.m.Next(s.stateFunctionName)
	case "=":
		// Assignment form: name = function ...
		s.pendingName = s.lastToken
	case "do", "then", "repeat":
		s.depth++
	case "end":
		if s.inFunc && s.depth == s.funcDepth {
			s.ctx.EndOfFunction()
			s.inFunc = false
		} else if s.depth > 0 {
			s.depth--
		}
	}
	return false
}

func (s *luaMachine) stateFunctionName(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "(" {
		// Anonymous function or assignment form
		name := s.pendingName
		if name == "" {
			name = "(anonymous)"
		}
		s.pendingName = ""
		s.ctx.PushNewFunction(name)
		s.m.Next(s.stateFunctionParams)
	} else {
		// Named function: `function foo(`
		s.pendingName = ""
		s.ctx.PushNewFunction(tok)
		s.m.Next(s.stateAfterFunctionName)
	}
	return false
}

func (s *luaMachine) stateAfterFunctionName(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case "(":
		s.m.Next(s.stateFunctionParams)
	case ".", ":":
		// Method syntax: function obj.method() or function obj:method()
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateMethodSuffix)
	}
	return false
}

func (s *luaMachine) stateMethodSuffix(tok string) bool {
	s.ctx.AddToFunctionName(tok)
	s.m.Next(s.stateAfterFunctionName)
	return false
}

func (s *luaMachine) stateFunctionParams(tok string) bool {
	defer func() { s.lastToken = tok }()
	switch tok {
	case ")":
		s.inFunc = true
		s.funcDepth = s.depth
		s.depth++
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
