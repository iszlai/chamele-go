// Package python implements the Python language reader.
package python

import (
	"iter"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() {
	languages.Register(NewPythonReader())
}

// PythonReader is the language reader for Python source files.
type PythonReader struct{}

func NewPythonReader() *PythonReader { return &PythonReader{} }

func (r *PythonReader) Extensions() []string    { return []string{"py"} }
func (r *PythonReader) LanguageNames() []string { return []string{"python"} }

// Tokenize produces Python tokens: adds single-line `#...` comments and
// triple-quoted docstring patterns to the base tokenizer.
func (r *PythonReader) Tokenize(src []byte) iter.Seq[string] {
	// Triple-quoted strings use lazy .*? (safe with the (?s) flag in buildPattern).
	// Single-line comments extend to end of line (with backslash continuation).
	untilEnd := `(?:\\\n|[^\n])*`
	addition := `|""".*?"""` +
		`|'''.*?'''` +
		`|#` + untilEnd
	return tokenizer.GenerateTokens(src, addition)
}

// GetComment returns the comment body for `#...` comment tokens and
// `#lizard forgive` directives.
func (r *PythonReader) GetComment(tok string) (string, bool) {
	if strings.HasPrefix(tok, "#") {
		body := strings.TrimLeft(tok, "#")
		body = strings.TrimSpace(body)
		return body, true
	}
	return "", false
}

// GetConditions returns the Python CCN condition token set.
func (r *PythonReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "elif": {}, "for": {}, "while": {}, "except": {}, "finally": {},
		"and": {}, "or": {},
	}
}

// Preprocess handles Python's indentation-based nesting and strips whitespace.
func (r *PythonReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		indents := &pythonIndents{ctx: ctx}
		currentSpaces := 0
		readingLeadingSpace := true

		for tok := range tokens {
			if tok != "\n" {
				if readingLeadingSpace {
					if stringx.IsHSpace(tok) {
						currentSpaces += countSpaces(tok)
					} else {
						if !strings.HasPrefix(tok, "#") {
							// Mirror Python: only update indentation when at global scope
							// or right after a function definition (long_name ends with ')').
							name := ctx.CurrentFunctionName()
							lname := ctx.CurrentFunctionLongName()
							if name == "*global*" || strings.HasSuffix(lname, ")") {
								indents.setNesting(currentSpaces, tok)
							}
						}
						readingLeadingSpace = false
						if !yield(tok) {
							return
						}
					}
				} else {
					if !stringx.IsHSpace(tok) {
						if !yield(tok) {
							return
						}
					}
				}
			} else {
				// Newline resets leading-space tracking.
				readingLeadingSpace = true
				currentSpaces = 0
				if !yield(tok) {
					return
				}
			}
		}
		indents.setNesting(0, "") // pop all remaining nesting levels
	}
}

// pythonIndents tracks Python's indentation-based nesting levels.
type pythonIndents struct {
	indents []int
	ctx     languages.Context
}

func (p *pythonIndents) setNesting(spaces int, tok string) {
	for len(p.indents) > 0 && p.indents[len(p.indents)-1] > spaces {
		if !strings.HasPrefix(tok, ")") {
			p.indents = p.indents[:len(p.indents)-1]
			p.ctx.PopNesting()
		} else {
			break
		}
	}
	if len(p.indents) == 0 || p.indents[len(p.indents)-1] < spaces {
		p.indents = append(p.indents, spaces)
		p.ctx.AddBareNesting()
	}
}

// RunTokens drives the Python state machine.
func (r *PythonReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	m := newPythonMachine(ctx)
	for tok := range tokens {
		m.Call(tok)
	}
}

// ---- Python state machine ----

type pythonMachine struct {
	m   *tokenizer.Machine
	ctx languages.Context
}

func newPythonMachine(ctx languages.Context) *tokenizer.Machine {
	s := &pythonMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *pythonMachine) stateGlobal(tok string) bool {
	if tok == "def" {
		s.m.Next(s.stateFunction)
	}
	return false
}

// stateFunction: reads the function name token, then waits for `(`.
func (s *pythonMachine) stateFunction(tok string) bool {
	if tok != "(" {
		s.ctx.RestartNewFunction(tok)
		s.ctx.AddToLongFunctionName("(")
	} else {
		s.m.Next(s.stateDec)
	}
	return false
}

// stateDec: reads parameters inside `(...)`.
func (s *pythonMachine) stateDec(tok string) bool {
	switch tok {
	case ")":
		s.ctx.AddToLongFunctionName(" )")
		s.m.Next(s.stateColon)
	case "[":
		s.ctx.AddToLongFunctionName(" " + tok)
		s.m.Next(s.stateAnnotationType)
	default:
		if !stringx.IsHSpace(tok) && tok != "\n" {
			s.ctx.Parameter(tok)
		}
		s.ctx.AddToLongFunctionName(" " + tok)
	}
	return false
}

// stateAnnotationType: reads inside `[...]` type annotation in parameter list.
func (s *pythonMachine) stateAnnotationType(tok string) bool {
	s.ctx.AddToLongFunctionName(" " + tok)
	if tok == "]" {
		s.m.Next(s.stateDec)
	}
	return false
}

// stateColon: after `)`, wait for `:` to confirm the function.
func (s *pythonMachine) stateColon(tok string) bool {
	if tok == ":" {
		s.m.Next(s.stateFirstLine)
	} else {
		s.m.Next(s.stateGlobal)
	}
	return false
}

// stateFirstLine: the first token inside the function body.
// If it's a docstring, subtract those NLOC.
func (s *pythonMachine) stateFirstLine(tok string) bool {
	s.m.Next(s.stateGlobal)
	if strings.HasPrefix(tok, `"""`) || strings.HasPrefix(tok, `'''`) {
		s.ctx.AddNLOC(-(strings.Count(tok, "\n") + 1))
	}
	s.stateGlobal(tok)
	return false
}

// addNLOC is used to adjust NLOC for docstrings. We add this to Context below.
func countSpaces(tok string) int {
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
