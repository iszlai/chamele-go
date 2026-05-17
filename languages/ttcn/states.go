package ttcn

import (
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

// newTTCNMachineConstructor creates the TTCN-3 function-detection state machine.
func newTTCNMachineConstructor(ctx languages.Context) *tokenizer.Machine {
	s := &ttcnMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

type ttcnMachine struct {
	m   *tokenizer.Machine
	ctx languages.Context
}

func (s *ttcnMachine) stateGlobal(tok string) bool {
	switch tok {
	case "function", "testcase", "altstep":
		s.ctx.TryNewFunction("")
		s.m.Next(s.stateFunctionName)
	case "control":
		s.ctx.TryNewFunction("__control__")
		s.m.Next(s.stateDecToImp)
	}
	return false
}

func (s *ttcnMachine) stateFunctionName(tok string) bool {
	if len(tok) > 0 && (isAlpha(tok[0]) || tok[0] == '_') {
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateExpectDec)
	} else if tok == "(" {
		s.m.Next(s.stateDec(), tok)
	} else {
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *ttcnMachine) stateExpectDec(tok string) bool {
	if tok == "(" {
		s.m.Next(s.stateDec(), tok)
	} else {
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *ttcnMachine) stateDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateDecToImp, func(tok string) {
		if tok != "(" && tok != ")" {
			s.ctx.Parameter(tok)
		}
		s.ctx.AddToLongFunctionName(" " + tok)
	})
}

func (s *ttcnMachine) stateDecToImp(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImp, "{")
	} else {
		// Skip return types, runs on etc.
		s.ctx.AddToLongFunctionName(" " + tok)
	}
	return false
}

func (s *ttcnMachine) stateEnteringImp(_ string) bool {
	s.ctx.ConfirmNewFunction()
	s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "{", "}", s.stateGlobal, func(_ string) {}), "{")
	return false
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
