// Package indented provides shared indentation-tracking machinery for
// indentation-sensitive languages (Python, GDScript, …).
//
// Indents tracks the stack of indentation levels seen so far and translates
// dedents into NestingStack.PopNesting calls. The caller's state machine
// consumes the post-preprocessed token stream and never sees raw indentation.
package indented

import (
	"strings"

	"github.com/iszlai/chamele-go/languages"
)

// Indents is the indentation-level stack. Construct with &Indents{Ctx: ctx}.
type Indents struct {
	Ctx    languages.Context
	levels []int
}

// SetNesting reconciles the current indentation (in spaces) with the
// recorded stack. Pops levels deeper than spaces; pushes a new level if
// spaces exceeds the top. The first token of the indented line is passed
// as tok so we can avoid popping on a continuation `)` that visually
// "outdents" but logically continues an open call.
func (p *Indents) SetNesting(spaces int, tok string) {
	for len(p.levels) > 0 && p.levels[len(p.levels)-1] > spaces {
		if strings.HasPrefix(tok, ")") {
			return
		}
		p.levels = p.levels[:len(p.levels)-1]
		p.Ctx.PopNesting()
	}
	if len(p.levels) == 0 || p.levels[len(p.levels)-1] < spaces {
		p.levels = append(p.levels, spaces)
		p.Ctx.AddBareNesting()
	}
}

// CountSpaces returns the visual column width of tok, where each tab is
// worth 8 spaces. tok is the leading-whitespace token produced by the
// tokenizer.
func CountSpaces(tok string) int {
	n := 0
	for _, c := range tok {
		if c == '\t' {
			n += 8
		} else {
			n++
		}
	}
	return n
}
