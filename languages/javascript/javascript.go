// Package javascript implements the JavaScript language reader.
package javascript

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() {
	languages.Register(NewJSReader())
}

// JSReader handles JavaScript source files.
type JSReader struct {
	*clike.CLikeReader
}

func NewJSReader() *JSReader {
	return &JSReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *JSReader) Extensions() []string    { return []string{"js", "cjs", "mjs"} }
func (r *JSReader) LanguageNames() []string { return []string{"javascript", "js"} }

const jsAddition = `|/(?:[^/\\\n]|\\.)+/[gimsuy]*` + // regex literals
	"|`[^`]*`" // template literals

func (r *JSReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, clike.CLikeAddition()+jsAddition)
}

func (r *JSReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {},
		"&&": {}, "||": {}, "case": {}, "?": {},
	}
}

// RunTokens drives only the JS machine. CLikeNestingStackStates is NOT used
// because JS manages its own brace-depth tracking for EndOfFunction calls.
func (r *JSReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newJSMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- JS state machine ----
//
// Design: function names are resolved at function-body `{` confirmation, then
// PushNewFunction saves the outer context. EndOfFunction is called when the
// matching `}` is seen via braceDepth tracking. This is identical to the Go
// reader approach and avoids conflicts with CLikeNestingStackStates.

type jsMachine struct {
	m             *tokenizer.Machine
	ctx           languages.Context
	bracketStack  []string
	pendingName   string // resolved LHS name: a=, a:, or function name
	pendingParams []string
	nameBuilder   strings.Builder
	dotMode       bool
	braceDepth    int
	funcDepths    []int // braceDepth before each function's { (for EndOfFunction)
}

func newJSMachine(ctx languages.Context) *tokenizer.Machine {
	s := &jsMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

// jsKeywords are identifiers that are NOT shorthand method names when followed by (.
var jsKeywords = map[string]bool{
	"if": true, "else": true, "for": true, "while": true, "switch": true,
	"catch": true, "try": true, "finally": true, "throw": true, "return": true,
	"typeof": true, "instanceof": true, "in": true, "of": true,
	"new": true, "delete": true, "void": true,
	"var": true, "let": true, "const": true,
	"class": true, "extends": true, "super": true, "this": true,
	"import": true, "export": true, "default": true, "from": true,
	"async": true, "await": true, "yield": true,
	"static": true, "get": true, "set": true,
	"do": true, "break": true, "continue": true, "debugger": true,
}

func (s *jsMachine) stateGlobal(tok string) bool {
	switch {
	case tok == "function":
		// Resolve the function name now; PushNewFunction happens at {.
		s.pendingName = s.resolvedName()
		s.nameBuilder.Reset()
		s.dotMode = false
		s.m.Next(s.stateFunctionName)

	case tok == "=":
		s.pendingName = s.nameBuilder.String()
		s.nameBuilder.Reset()
		s.dotMode = false

	case tok == ":":
		// Object key assignment
		s.pendingName = s.nameBuilder.String()
		s.nameBuilder.Reset()
		s.dotMode = false

	case tok == "(":
		// Shorthand method: name() { body } — only if name is not a JS keyword.
		name := s.nameBuilder.String()
		if name != "" && !jsKeywords[name] {
			s.pendingName = name
			s.bracketStack = nil
			s.m.Next(s.stateDecFn(), tok)
		}
		// Don't reset nameBuilder — let default handle it on non-match.

	case tok == ".":
		s.dotMode = true

	case tok == "{":
		s.braceDepth++
		s.nameBuilder.Reset()
		s.pendingName = ""
		s.dotMode = false

	case tok == "}":
		s.braceDepth--
		if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
			s.ctx.EndOfFunction()
			s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
		}

	case len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_' || tok[0] == '$'):
		if s.dotMode {
			s.nameBuilder.WriteString("." + tok)
			s.dotMode = false
		} else {
			s.nameBuilder.Reset()
			s.nameBuilder.WriteString(tok)
			s.dotMode = false
		}

	default:
		if tok != "(" && tok != ")" && tok != "," && tok != ";" {
			s.nameBuilder.Reset()
			s.pendingName = ""
			s.dotMode = false
		}
	}
	return false
}

func (s *jsMachine) resolvedName() string {
	if s.pendingName != "" {
		return s.pendingName
	}
	return s.nameBuilder.String()
}

// stateFunctionName: next token is either the function name or `(` (anonymous).
func (s *jsMachine) stateFunctionName(tok string) bool {
	switch {
	case tok == "(":
		if s.pendingName == "" {
			s.pendingName = "(anonymous)"
		}
		s.bracketStack = nil
		s.m.Next(s.stateDecFn(), tok)
	case len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_'):
		// Named function: the name is appended to any pending LHS name.
		if s.pendingName == "" {
			s.pendingName = tok
		} else {
			s.pendingName = tok // explicit function name overrides LHS
		}
		s.m.Next(s.stateExpectDec)
	default:
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *jsMachine) stateExpectDec(tok string) bool {
	if tok == "(" {
		s.bracketStack = nil
		s.m.Next(s.stateDecFn(), tok)
	} else {
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *jsMachine) stateDecFn() tokenizer.StateFn {
	s.pendingParams = nil
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateDecToImp, func(tok string) {
		switch {
		case tok == "(" || tok == "<":
			s.bracketStack = append(s.bracketStack, tok)
		case tok == ")" || tok == ">":
			if len(s.bracketStack) > 0 {
				s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			}
		case len(s.bracketStack) == 1:
			if tok != "void" && tok != "," {
				s.pendingParams = append(s.pendingParams, tok)
			} else if tok == "," {
				s.pendingParams = append(s.pendingParams, ",")
			}
		}
	})
}

func (s *jsMachine) stateDecToImp(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImp, "{")
	case "=>":
		s.m.Next(s.stateArrow)
	default:
		// Not a function body (call, expression, etc.) — abandon and re-process.
		s.pendingName = ""
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *jsMachine) stateArrow(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImp, "{")
	} else {
		// Expression arrow — push a minimal function record.
		s.ctx.PushNewFunction(s.pendingName)
		s.ctx.EndOfFunction()
		s.pendingName = ""
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *jsMachine) stateEnteringImp(_ string) bool {
	name := s.pendingName
	if name == "" {
		name = "(anonymous)"
	}
	s.ctx.PushNewFunction(name)
	// Apply parameters accumulated during stateDecFn.
	for _, p := range s.pendingParams {
		s.ctx.Parameter(p)
	}
	s.pendingParams = nil
	s.pendingName = ""
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}
