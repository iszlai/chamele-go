// Package solidity implements the Solidity language reader.
package solidity

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewSolidityReader()) }

// SolidityReader handles Solidity smart contract source files (.sol).
// Solidity uses `function name(...) {}`, `modifier name(...){}`, and `constructor(...){}`.
type SolidityReader struct {
	*clike.CLikeReader
}

func NewSolidityReader() *SolidityReader {
	return &SolidityReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *SolidityReader) Extensions() []string    { return []string{"sol"} }
func (r *SolidityReader) LanguageNames() []string { return []string{"solidity"} }

func (r *SolidityReader) Tokenize(src []byte) iter.Seq[string] {
	return r.CLikeReader.Tokenize(src)
}

func (r *SolidityReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "&&": {}, "||": {}, "?": {},
	}
}

func (r *SolidityReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newSolidityMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

type solidityMachine struct {
	m             *tokenizer.Machine
	ctx           languages.Context
	braceDepth    int
	funcDepths    []int
	pendingName   string
	pendingParams []string
	bracketStack  []string
}

func newSolidityMachine(ctx languages.Context) *tokenizer.Machine {
	s := &solidityMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *solidityMachine) stateGlobal(tok string) bool {
	switch tok {
	case "function", "modifier", "constructor":
		s.pendingName = ""
		s.pendingParams = nil
		s.m.Next(s.stateFunctionName)
	case "{":
		s.braceDepth++
	case "}":
		s.braceDepth--
		if len(s.funcDepths) > 0 && s.braceDepth == s.funcDepths[len(s.funcDepths)-1] {
			s.ctx.EndOfFunction()
			s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
		}
	}
	return false
}

func (s *solidityMachine) stateFunctionName(tok string) bool {
	switch tok {
	case "(":
		s.bracketStack = nil
		s.m.Next(s.stateFunctionDec(), tok)
	case "{":
		s.m.Next(s.stateEnteringImpl, tok)
	case ";":
		s.pendingName = ""
		s.m.Next(s.stateGlobal)
	default:
		if len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_') {
			s.pendingName = tok
		}
	}
	return false
}

func (s *solidityMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectImpl, func(tok string) {
		switch {
		case tok == "(" || tok == "<":
			s.bracketStack = append(s.bracketStack, tok)
		case tok == ")" || tok == ">":
			if len(s.bracketStack) > 0 {
				s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			}
		case len(s.bracketStack) == 1 && tok != ",":
			s.pendingParams = append(s.pendingParams, tok)
		}
	})
}

func (s *solidityMachine) stateExpectImpl(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImpl, tok)
	case ";":
		s.pendingName = ""
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *solidityMachine) stateEnteringImpl(_ string) bool {
	name := s.pendingName
	if name == "" {
		name = "(anonymous)"
	}
	s.ctx.PushNewFunction(name)
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
