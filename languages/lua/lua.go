// Package lua implements the Lua language reader.
package lua

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
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
// Uses a block stack: true = function block, false = regular block (do/then/repeat).
// 'end' pops the stack and calls EndOfFunction when a function block is popped.
// LineCounter swallows \n before RunTokens, so \n-based triggers are unreliable;
// instead, any unexpected token after a function name starts the body.

type luaMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	blockStack  []bool // true=function, false=regular
	pendingName string // name from LHS of assignment
	lastToken   string
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
		s.m.Next(s.stateFunctionName)
	case "=":
		s.pendingName = s.lastToken
	case "do", "repeat", "if":
		// 'do' opens for/while/standalone blocks; 'repeat' opens repeat..until;
		// 'if' opens if..elseif..else..end (elseif/else don't open new blocks).
		s.blockStack = append(s.blockStack, false)
	case "until":
		// Closes a repeat..until block (no 'end' for repeat).
		if len(s.blockStack) > 0 && !s.blockStack[len(s.blockStack)-1] {
			s.blockStack = s.blockStack[:len(s.blockStack)-1]
		}
	case "end":
		if len(s.blockStack) > 0 {
			isFunc := s.blockStack[len(s.blockStack)-1]
			s.blockStack = s.blockStack[:len(s.blockStack)-1]
			if isFunc {
				s.ctx.EndOfFunction()
			}
		}
	}
	return false
}

func (s *luaMachine) stateFunctionName(tok string) bool {
	defer func() { s.lastToken = tok }()
	if tok == "(" {
		// Anonymous function or `name = function(` assignment form
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
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateMethodSuffix)
	default:
		// Start function body. LineCounter swallows \n before RunTokens so we
		// never see the header-ending newline; the first body token arrives here.
		s.blockStack = append(s.blockStack, true)
		s.m.Next(s.stateGlobal, tok)
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
		s.blockStack = append(s.blockStack, true)
		s.m.Next(s.stateGlobal)
	case ",":
		// separator
	default:
		if !stringx.IsHSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
		}
	}
	return false
}
