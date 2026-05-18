// Package duplicate implements a (simplified) duplicate-code-block detector.
//
// Each function's tokens are unified — identifiers are renamed to canonical
// 'v0','v1',… per scope, numeric and string literals are collapsed to '1',
// '-' is folded into '+' — and then sliced into fixed-size windows of
// SampleSize tokens. Windows that hash identically across (or within)
// functions are reported as duplicates.
//
// This is a deliberately simpler scheme than upstream lizard's
// lizardduplicate.py, which uses a rolling boundary-aware sequence
// extender. The chamele-go version is meant to surface the obvious cases.
package duplicate

import (
	"fmt"
	"io"
	"iter"
	"sort"
	"strings"
	"unicode"

	"github.com/iszlai/chamele-go/chamele"
)

const (
	// SampleSize is the number of tokens per duplicate-detection window.
	SampleSize = 31

	tokensKey = "duplicate_tokens"
	linesKey  = "duplicate_lines"
)

func init() { chamele.RegisterExtension(New()) }

// New returns the duplicate extension instance.
func New() chamele.Extension { return ext }

var ext = &dupExt{}

type dupExt struct {
	dups               []dupBlock
	totalTokens        int
	uniqueTokens       int
	savedDuplicateRate float64
	savedUniqueRate    float64
}

type dupBlock struct {
	Files     []string
	Start     int
	EndLine   int
	StartLine int
}

func (e *dupExt) Name() string       { return "duplicate" }
func (e *dupExt) OrderingIndex() int { return 1000 }

func (e *dupExt) FunctionInfoColumns() []chamele.ColumnSpec { return nil }

// Process records every token (with the current line) so the cross-file
// pass can rebuild and hash unified windows.
func (e *dupExt) Process(tokens iter.Seq[string], ctx *chamele.FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			fn := ctx.CurrentFunction
			if fn.Ext == nil {
				fn.Ext = make(map[string]any)
			}
			toks, _ := fn.Ext[tokensKey].([]string)
			lines, _ := fn.Ext[linesKey].([]int)
			fn.Ext[tokensKey] = append(toks, tok)
			fn.Ext[linesKey] = append(lines, ctx.CurrentLine)
			if !yield(tok) {
				return
			}
		}
	}
}

// CrossFileProcess builds per-file unified token windows, hashes them, and
// records hash windows that appear in more than one place.
func (e *dupExt) CrossFileProcess(files []chamele.FileInformation) []chamele.FileInformation {
	type loc struct {
		file      string
		startLine int
		endLine   int
	}
	hashToLocs := map[string][]loc{}

	for i := range files {
		filename := files[i].Filename
		for _, fn := range files[i].Functions {
			toks, _ := fn.Ext[tokensKey].([]string)
			lines, _ := fn.Ext[linesKey].([]int)
			e.totalTokens += len(toks)

			unified := unify(toks)
			if len(unified) < SampleSize {
				continue
			}
			for start := 0; start+SampleSize <= len(unified); start++ {
				h := strings.Join(unified[start:start+SampleSize], "|")
				sLine := lines[start]
				eLine := lines[start+SampleSize-1]
				hashToLocs[h] = append(hashToLocs[h], loc{filename, sLine, eLine})
			}
		}
	}

	e.uniqueTokens = len(hashToLocs)
	for _, locs := range hashToLocs {
		if len(locs) < 2 {
			continue
		}
		fileSet := map[string]struct{}{}
		var fnames []string
		minStart, maxEnd := locs[0].startLine, locs[0].endLine
		for _, l := range locs {
			if _, ok := fileSet[l.file]; !ok {
				fileSet[l.file] = struct{}{}
				fnames = append(fnames, l.file)
			}
			if l.startLine < minStart {
				minStart = l.startLine
			}
			if l.endLine > maxEnd {
				maxEnd = l.endLine
			}
		}
		sort.Strings(fnames)
		e.dups = append(e.dups, dupBlock{
			Files:     fnames,
			StartLine: minStart,
			EndLine:   maxEnd,
		})
	}

	if e.totalTokens > 0 {
		e.savedDuplicateRate = float64(len(e.dups)*SampleSize) / float64(e.totalTokens)
		e.savedUniqueRate = float64(e.uniqueTokens) / float64(e.totalTokens)
	}

	return files
}

// PrintResult writes a summary of detected duplicate blocks to w.
func (e *dupExt) PrintResult(w io.Writer) error {
	if len(e.dups) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "Duplicates"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "==================================="); err != nil {
		return err
	}
	for _, d := range e.dups {
		if _, err := fmt.Fprintln(w, "Duplicate block:"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, "--------------------------"); err != nil {
			return err
		}
		for _, f := range d.Files {
			if _, err := fmt.Fprintf(w, "%s: lines %d~%d\n", f, d.StartLine, d.EndLine); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w, "^^^^^^^^^^^^^^^^^^^^^^^^^^"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "Total duplicate rate: %.2f%%\n", e.savedDuplicateRate*100); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "Total unique rate: %.2f%%\n", e.savedUniqueRate*100)
	return err
}

// unify renames identifiers and collapses literals to canonical forms so
// that two structurally equivalent blocks hash identically.
func unify(toks []string) []string {
	out := make([]string, 0, len(toks))
	register := map[string]string{}
	counter := 0
	prev := ""
	for _, t := range toks {
		if prev == "." || prev == "->" {
			out = append(out, t)
			prev = t
			continue
		}
		if t == "-" {
			out = append(out, "+")
			prev = "+"
			continue
		}
		if t == "" {
			prev = t
			out = append(out, t)
			continue
		}
		first := rune(t[0])
		if unicode.IsDigit(first) || first == '\'' || first == '"' {
			out = append(out, "1")
			prev = "1"
			continue
		}
		if unicode.IsLetter(first) || first == '_' {
			v, ok := register[t]
			if !ok {
				v = fmt.Sprintf("v%d", counter)
				counter++
				register[t] = v
			}
			out = append(out, v)
			prev = v
			continue
		}
		out = append(out, t)
		prev = t
	}
	return out
}
