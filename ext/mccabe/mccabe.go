// Package mccabe implements the strict McCabe CCN extension.
//
// In McCabe's original definition (http://www.mccabe.com/pdf/mccabe-nist235r.pdf)
// consecutive switch/case fall-through labels do not each contribute +1 to
// cyclomatic complexity — only the first 'case' counts; subsequent cases
// with no code between them are folded into the previous branch.
//
// Derived from upstream lizard_ext/lizardmccabe.py.
package mccabe

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

// New returns the McCabe extension instance.
func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string                              { return "mccabe" }
func (e *ext) OrderingIndex() int                        { return 1100 } // run after ConditionCounter
func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

type state int

const (
	stateGlobal state = iota
	stateInCase
	stateAfterCase
)

func (e *ext) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	st := stateGlobal
	return func(yield func(string) bool) {
		for tok := range tokens {
			switch st {
			case stateGlobal:
				if tok == "case" {
					st = stateInCase
				}
			case stateInCase:
				if tok == ":" {
					st = stateAfterCase
				}
			case stateAfterCase:
				if tok == "case" {
					// Consecutive case label without intervening code:
					// cancel the +1 ConditionCounter added.
					ctx.AddCondition(-1)
					st = stateInCase
				} else {
					st = stateGlobal
					if tok == "case" {
						st = stateInCase
					}
				}
			}
			if !yield(tok) {
				return
			}
		}
	}
}
