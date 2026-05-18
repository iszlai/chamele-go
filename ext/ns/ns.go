// Package ns implements the nested-structures (NS) extension.
//
// It counts the maximum number of nested control structures within each
// function. Derived from upstream lizard_ext/lizardns.py. The
// implementation tracks a stack of structure counts per scope and updates
// FunctionInfo.Ext["max_nested_structures"] on every increment.
package ns

import (
	"iter"

	"github.com/iszlai/chamele-go/chamele"
)

// Key is the FunctionInfo.Ext map key for the max-nested-structures metric.
const Key = "max_nested_structures"

var structures = map[string]struct{}{
	"if": {}, "else": {}, "elif": {}, "for": {}, "foreach": {},
	"while": {}, "do": {}, "try": {}, "catch": {}, "switch": {},
	"finally": {}, "except": {}, "with": {},
}

var matchingStructures = map[string]struct{}{
	"else": {}, "elif": {}, "catch": {}, "finally": {},
}

func init() { chamele.RegisterExtension(New()) }

type ext struct{}

// New returns the NS extension instance.
func New() chamele.Extension { return &ext{} }

func (e *ext) Name() string       { return "ns" }
func (e *ext) OrderingIndex() int { return 1000 }

func (e *ext) FunctionInfoColumns() []chamele.ColumnSpec {
	return []chamele.ColumnSpec{{
		Header: "  NS  ",
		Value:  func(f *chamele.FunctionInfo) any { return f.GetInt(Key) },
	}}
}

type state int

const (
	stateGlobal state = iota
	stateInHead
	stateBlockEnd
)

func (e *ext) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	piles := []int{0}
	st := stateGlobal
	parenDepth := 0

	push := func() { piles = append(piles, 0) }
	pop := func() {
		if len(piles) > 1 {
			piles = piles[:len(piles)-1]
		}
	}
	sum := func() int {
		s := 0
		for _, p := range piles {
			s += p
		}
		return s
	}

	pileUp := func(fn *chamele.FunctionInfo) {
		piles[len(piles)-1]++
		cur := sum()
		if cur > fn.GetInt(Key) {
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			fn.Ext[Key] = cur
		}
	}

	feedGlobal := func(tok string) {
		switch tok {
		case "{":
			push()
		case "}":
			pop()
			st = stateBlockEnd
		case ";":
			st = stateBlockEnd
		default:
			if _, isStruct := structures[tok]; isStruct {
				st = stateInHead
				parenDepth = 0
			}
		}
	}

	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction

			switch st {
			case stateGlobal:
				feedGlobal(tok)
			case stateInHead:
				switch tok {
				case "(":
					parenDepth++
				case ")":
					parenDepth--
					if parenDepth <= 0 {
						pileUp(fn)
						st = stateGlobal
					}
				default:
					if parenDepth == 0 {
						pileUp(fn)
						st = stateGlobal
						feedGlobal(tok)
					}
				}
			case stateBlockEnd:
				if _, matching := matchingStructures[tok]; matching {
					if piles[len(piles)-1] > 0 {
						piles[len(piles)-1]--
					}
				} else {
					piles[len(piles)-1] = 0
				}
				st = stateGlobal
				feedGlobal(tok)
			}

			if !yield(tok) {
				return
			}
		}
	}
}
