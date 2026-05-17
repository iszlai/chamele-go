package tokenizer

import "iter"

// Conditions is the set of tokens that each increment the cyclomatic complexity
// counter by 1 when encountered.
type Conditions map[string]struct{}

// Has reports whether tok is in the condition set.
func (c Conditions) Has(tok string) bool {
	_, ok := c[tok]
	return ok
}

// DefaultConditions returns the base condition set used by C-like languages.
func DefaultConditions() Conditions {
	return Conditions{
		"if": {}, "for": {}, "while": {}, "catch": {},
		"&&": {}, "||": {},
		"case": {},
		"?":    {},
	}
}

// BaseReader holds the fields common to all language readers: the set of
// parallel state machines and the condition token set.
//
// Language readers embed BaseReader and call RunParallelStates in their token
// processing loop.
type BaseReader struct {
	ParallelStates      []*Machine
	Conditions          Conditions
	ControlFlowKeywords Conditions
	LogicalOperators    Conditions
	CaseKeywords        Conditions
	TernaryOperators    Conditions
}

// NewBaseReader initialises a BaseReader with the default C-like condition sets.
func NewBaseReader() BaseReader {
	cf := Conditions{"if": {}, "for": {}, "while": {}, "catch": {}}
	lo := Conditions{"&&": {}, "||": {}}
	ck := Conditions{"case": {}}
	to := Conditions{"?": {}}

	combined := Conditions{}
	for k := range cf {
		combined[k] = struct{}{}
	}
	for k := range lo {
		combined[k] = struct{}{}
	}
	for k := range ck {
		combined[k] = struct{}{}
	}
	for k := range to {
		combined[k] = struct{}{}
	}

	return BaseReader{
		Conditions:          combined,
		ControlFlowKeywords: cf,
		LogicalOperators:    lo,
		CaseKeywords:        ck,
		TernaryOperators:    to,
	}
}

// RunParallelStates feeds tok to every machine in ParallelStates.
func (br *BaseReader) RunParallelStates(tok string) {
	for _, m := range br.ParallelStates {
		m.Call(tok)
	}
}

// EOF notifies all parallel state machines that the token stream has ended.
func (br *BaseReader) EOF() {
	for _, m := range br.ParallelStates {
		m.StatemachineBeforeReturn()
	}
}

// FilterTokens removes whitespace tokens (except bare newlines) from tokens,
// mirroring Python's preprocessing() default behaviour.
func FilterTokens(tokens iter.Seq[string]) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "\n" || !isSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}

func isSpace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' {
			return false
		}
	}
	return len(s) > 0
}
