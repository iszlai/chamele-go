// Package php implements the PHP language reader.
package php

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() { languages.Register(NewPHPReader()) }

// PHPReader handles PHP source files (.php).
// PHP uses the `function` keyword, and code lives inside <?php ... ?> blocks.
type PHPReader struct {
	*clike.CLikeReader
}

func NewPHPReader() *PHPReader {
	return &PHPReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *PHPReader) Extensions() []string    { return []string{"php"} }
func (r *PHPReader) LanguageNames() []string { return []string{"php"} }

func (r *PHPReader) Tokenize(src []byte) iter.Seq[string] {
	// Add PHP variable pattern $name and heredoc-like strings, plus C-like patterns.
	return tokenizer.GenerateTokens(src, clike.CLikeAddition()+`|(?:\$\w+)`)
}

// GetConditions returns PHP CCN conditions.
func (r *PHPReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elseif": {}, "for": {}, "foreach": {}, "while": {}, "catch": {}, "case": {},
		"&&": {}, "||": {}, "?": {},
	}
}

// Preprocess extracts code from inside <?php ... ?> blocks and strips whitespace.
// The tokenizer produces `<`, `?`, `php` as separate tokens for `<?php`.
func (r *PHPReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		inPHP := false
		prevWasLT := false // was previous token '<'
		prevWasQ := false  // was previous token '?'
		for tok := range tokens {
			// Detect the sequence <, ?, php (or just <?) to enter PHP mode.
			if tok == "<" {
				prevWasLT = true
				prevWasQ = false
				continue
			}
			if prevWasLT && tok == "?" {
				prevWasLT = false
				prevWasQ = true
				continue
			}
			if prevWasQ {
				prevWasQ = false
				// The token after < ? is either "php", "=" or nothing (short open)
				if strings.EqualFold(tok, "php") || tok == "=" {
					inPHP = true
					continue
				}
				// otherwise treat as PHP already open (short tag)
				inPHP = true
				// fall through to emit tok
			}
			prevWasLT = false

			// Detect ?> to leave PHP mode (also tokenized as '?', '>')
			if inPHP && tok == "?" {
				prevWasQ = true
				continue
			}
			if prevWasQ && tok == ">" {
				prevWasQ = false
				inPHP = false
				continue
			}
			if prevWasQ {
				// The '?' was not part of ?>; emit it if we're in PHP
				if inPHP {
					if !yield("?") {
						return
					}
				}
				prevWasQ = false
			}

			if !inPHP {
				continue
			}
			// Strip horizontal whitespace, keep newlines.
			if tok == "\n" || !stringx.IsHSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}

// RunTokens drives the PHP state machine using the JS-style brace-depth tracker
// which is appropriate for PHP's C-like function syntax.
func (r *PHPReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newPHPMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- PHP state machine ----

type phpMachine struct {
	m           *tokenizer.Machine
	ctx         languages.Context
	braceDepth  int
	funcDepths  []int
	pendingName string
}

func newPHPMachine(ctx languages.Context) *tokenizer.Machine {
	s := &phpMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *phpMachine) stateGlobal(tok string) bool {
	switch tok {
	case "function":
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

func (s *phpMachine) stateFunctionName(tok string) bool {
	if tok == "(" {
		// Anonymous function
		if s.pendingName == "" {
			s.pendingName = "(anonymous)"
		}
		s.ctx.PushNewFunction(s.pendingName)
		s.pendingName = ""
		s.m.Next(s.stateFunctionDec(), tok)
	} else if len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_') {
		s.ctx.PushNewFunction(tok)
		s.m.Next(s.stateExpectDec)
	}
	return false
}

func (s *phpMachine) stateExpectDec(tok string) bool {
	if tok == "(" {
		s.m.Next(s.stateFunctionDec(), tok)
	} else {
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *phpMachine) stateFunctionDec() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateExpectImpl, func(tok string) {
		if strings.HasPrefix(tok, "$") {
			s.ctx.Parameter(tok)
		}
	})
}

func (s *phpMachine) stateExpectImpl(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImpl, "{")
	case ";":
		// Abstract / interface method declaration
		s.ctx.EndOfFunction()
		s.m.Next(s.stateGlobal)
	case ":":
		// Return type hint, skip until {
	}
	return false
}

func (s *phpMachine) stateEnteringImpl(_ string) bool {
	s.funcDepths = append(s.funcDepths, s.braceDepth)
	s.braceDepth++
	s.m.Next(s.stateGlobal)
	return false
}
