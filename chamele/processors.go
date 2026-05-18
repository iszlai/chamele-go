package chamele

import (
	"iter"
	"regexp"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/languages"
)

// Processor is a transform stage in the analysis pipeline. It wraps the
// incoming token sequence in a new sequence and may accumulate state into ctx.
// reader is the language reader; processors type-assert it to optional
// capability interfaces (commenter, preprocessor, conditionProvider).
type Processor func(tokens iter.Seq[string], ctx *FileInfoBuilder, reader any) iter.Seq[string]

// --- optional reader capability interfaces ---

type commenter interface {
	GetComment(tok string) (string, bool)
}

type preprocessor interface {
	Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string]
}

type conditionProvider interface {
	GetConditions() map[string]struct{}
}

// --- the five standard processors, in pipeline order ---

// Preprocessing strips horizontal whitespace tokens unless the reader has a
// custom Preprocess method (e.g. for Python indentation tracking).
func Preprocessing(tokens iter.Seq[string], ctx *FileInfoBuilder, reader any) iter.Seq[string] {
	if p, ok := reader.(preprocessor); ok {
		return p.Preprocess(tokens, ctx)
	}
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "\n" || !stringx.IsHSpace(tok) {
				if !yield(tok) {
					return
				}
			}
		}
	}
}

// CommentCounter intercepts comment tokens, emits synthetic \n tokens for
// embedded newlines (to keep line counts correct), and handles forgiveness
// directives. If a comment contains "GENERATED CODE" the file is abandoned.
func CommentCounter(tokens iter.Seq[string], ctx *FileInfoBuilder, reader any) iter.Seq[string] {
	cr, hasComments := reader.(commenter)
	return func(yield func(string) bool) {
		for tok := range tokens {
			if hasComments {
				comment, isComment := cr.GetComment(tok)
				if isComment {
					// Emit a synthetic \n for each embedded newline past the first line.
					lines := strings.Split(comment, "\n")
					for range lines[1:] {
						if !yield("\n") {
							return
						}
					}
					stripped := strings.TrimSpace(comment)
					switch {
					case strings.HasPrefix(stripped, "#lizard forgive global"):
						ctx.ForgiveGlobal = true
					case strings.HasPrefix(stripped, "#lizard forgives("):
						if m := forgiveRe.FindStringSubmatch(stripped); m != nil {
							if ctx.CurrentFunction.ForgivenMetrics == nil {
								ctx.CurrentFunction.ForgivenMetrics = make(map[string]struct{})
							}
							for _, metric := range strings.Split(m[1], ",") {
								metric = strings.TrimSpace(metric)
								if metric != "" {
									ctx.CurrentFunction.ForgivenMetrics[metric] = struct{}{}
								}
							}
						}
					case strings.HasPrefix(stripped, "#lizard forgive"):
						ctx.Forgive = true
					}
					if strings.Contains(comment, "GENERATED CODE") {
						return // abandon this file
					}
					continue // comment consumed, do not yield tok
				}
			}
			if !yield(tok) {
				return
			}
		}
	}
}

var forgiveRe = regexp.MustCompile(`#lizard forgives?\(([^)]*)\)`)

// LineCounter tracks newlines to maintain CurrentLine and NLOC counts.
// Tokens that contain embedded newlines (multi-line strings, block comments
// already passed through as single tokens) contribute multiple NLOC.
func LineCounter(tokens iter.Seq[string], ctx *FileInfoBuilder, _ any) iter.Seq[string] {
	ctx.CurrentLine = 1
	newline := 1
	return func(yield func(string) bool) {
		for tok := range tokens {
			if tok == "\n" {
				ctx.CurrentLine++
				newline = 1
			} else {
				count := strings.Count(tok, "\n")
				ctx.CurrentLine += count
				ctx.AddNLOC(count + newline)
				newline = 0
				if !yield(tok) {
					return
				}
			}
		}
	}
}

// TokenCounter increments per-file and per-function token counts.
func TokenCounter(tokens iter.Seq[string], ctx *FileInfoBuilder, _ any) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			ctx.fileinfo.TokenCount++
			ctx.CurrentFunction.TokenCount++
			if !yield(tok) {
				return
			}
		}
	}
}

// ConditionCounter increments cyclomatic complexity for each condition token.
// The condition set is obtained from the reader if it implements conditionProvider;
// otherwise the default C-like set is used.
func ConditionCounter(tokens iter.Seq[string], ctx *FileInfoBuilder, reader any) iter.Seq[string] {
	var conds map[string]struct{}
	if cp, ok := reader.(conditionProvider); ok {
		conds = cp.GetConditions()
	} else {
		conds = defaultConditions
	}
	return func(yield func(string) bool) {
		for tok := range tokens {
			if _, ok := conds[tok]; ok {
				ctx.AddCondition(1)
			}
			if !yield(tok) {
				return
			}
		}
	}
}

var defaultConditions = map[string]struct{}{
	"if": {}, "for": {}, "while": {}, "catch": {},
	"&&": {}, "||": {},
	"case": {},
	"?":    {},
}

// DefaultProcessors returns the standard pipeline in lizard's defined order.
func DefaultProcessors() []Processor {
	return []Processor{
		Preprocessing,
		CommentCounter,
		LineCounter,
		TokenCounter,
		ConditionCounter,
	}
}
