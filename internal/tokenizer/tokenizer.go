// Package tokenizer implements the language-agnostic regex tokenizer and
// CodeStateMachine engine ported from lizard's code_reader.py.
package tokenizer

import (
	"iter"
	"regexp"
	"strings"
)

// combinedSymbols is the ordered list of multi-char operators, longest first
// within each group, matching lizard's generate_tokens ordering exactly.
var combinedSymbols = []string{
	"<<=", ">>=", "||", "&&", "===", "!==",
	"==", "!=", "<=", ">=", "->", "=>",
	"++", "--", "+=", "-=",
	"+", "-", "*", "/",
	"*=", "/=", "^=", "&=", "|=", "...",
}

// flagRe extracts inline flag groups like (?i) from language additions.
var flagRe = regexp.MustCompile(`\(\?[aiLmsux]+\)`)

// GenerateTokens tokenizes src using the base pattern plus any language-specific
// addition. The addition may contain inline flags like (?i) which are extracted
// and applied to the compiled pattern.
func GenerateTokens(src []byte, addition string) iter.Seq[string] {
	re := buildPattern(addition)
	return tokenize(string(src), re)
}

func buildPattern(addition string) *regexp.Regexp {
	// Extract inline flags from addition (e.g. "(?i)" → case-insensitive).
	flags := ""
	for _, m := range flagRe.FindAllString(addition, -1) {
		flags += m[2 : len(m)-1]
	}
	cleaned := flagRe.ReplaceAllString(addition, "")

	// Base: (?ms) = multiline + dotall. Append (?i) if addition requested it.
	prefix := "(?ms)"
	if strings.Contains(flags, "i") {
		prefix = "(?msi)"
	}

	// Line-comment tail: anything up to (but not including) a bare newline,
	// allowing \<backslash><newline> as a continuation.
	untilEnd := `(?:\\\n|[^\n])*`

	syms := make([]string, len(combinedSymbols))
	for i, s := range combinedSymbols {
		syms[i] = regexp.QuoteMeta(s)
	}

	// Strip a leading | from the addition: callers often write "|pattern"
	// to match Python's literal-concatenation style, but buildPattern
	// assembles parts with Join(parts, "|") so a leading | would produce "||".
	if len(cleaned) > 0 && cleaned[0] == '|' {
		cleaned = cleaned[1:]
	}

	parts := []string{`/\*.*?\*/`} // block comment first
	if cleaned != "" {
		parts = append(parts, cleaned)
	}
	parts = append(parts,
		`(?:\d+')+\d+`,                     // C++14 digit separators: 1'000'000
		`0x(?:[0-9A-Fa-f]+')+[0-9A-Fa-f]+`, // hex with separators
		`0b(?:[01]+')+[01]+`,               // binary with separators
		`\w+`,                              // identifiers and keywords
		`"(?:\\.|[^"\\])*"`,                // double-quoted strings
		`'(?:\\.|[^'\\])*?'`,               // single-quoted strings
		`//`+untilEnd,                      // line comments
		`#`,                                // macro start (collected below)
		`:=|::|\*\*`,                       // special operators
		strings.Join(syms, "|"),            // combined symbols
		`\\\n`,                             // line continuation (2-char token)
		`\n`,                               // newline
		`[^\S\n]+`,                         // horizontal whitespace
		`.`,                                // any other single character
	)

	pattern := prefix + "(?:" + strings.Join(parts, "|") + ")"
	return regexp.MustCompile(pattern)
}

// tokenize produces a token sequence from src using the compiled pattern.
// Tokens starting with '#' are accumulated into macro tokens, continuing
// across \<backslash><newline> line continuations and stopping at a bare \n.
func tokenize(src string, re *regexp.Regexp) iter.Seq[string] {
	return func(yield func(string) bool) {
		macro := ""
		for _, loc := range re.FindAllStringIndex(src, -1) {
			tok := src[loc[0]:loc[1]]
			if macro != "" {
				if strings.Contains(tok, "\\\n") || !strings.Contains(tok, "\n") {
					macro += tok
				} else {
					if !yield(macro) {
						return
					}
					if !yield(tok) {
						return
					}
					macro = ""
				}
			} else if tok == "#" {
				macro = tok
			} else {
				if !yield(tok) {
					return
				}
			}
		}
		if macro != "" {
			yield(macro)
		}
	}
}
