package tokenizer

import (
	"slices"
	"testing"
)

// collect drains a token sequence into a slice.
func collect(src string) []string {
	var out []string
	for tok := range GenerateTokens([]byte(src), "") {
		out = append(out, tok)
	}
	return out
}

func TestUnit_GenerateTokens_EmptyString(t *testing.T) {
	if got := collect(""); len(got) != 0 {
		t.Errorf("expected no tokens, got %v", got)
	}
}

func TestUnit_GenerateTokens_Spaces(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{"\n", []string{"\n"}},
		{"\n\n", []string{"\n", "\n"}},
		{" \n", []string{" ", "\n"}},
	}
	for _, c := range cases {
		if got := collect(c.src); !slices.Equal(got, c.want) {
			t.Errorf("collect(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestUnit_GenerateTokens_Digits(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{"1", []string{"1"}},
		{"123", []string{"123"}},
	}
	for _, c := range cases {
		if got := collect(c.src); !slices.Equal(got, c.want) {
			t.Errorf("collect(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestUnit_GenerateTokens_Operators(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{"-;", []string{"-", ";"}},
		{"-=", []string{"-="}},
		{">=", []string{">="}},
		{"<=", []string{"<="}},
		{"||", []string{"||"}},
		{">>", []string{">", ">"}},
		{">>=", []string{">>="}},
		{"<<=", []string{"<<="}},
	}
	for _, c := range cases {
		if got := collect(c.src); !slices.Equal(got, c.want) {
			t.Errorf("collect(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestUnit_GenerateTokens_Identifiers(t *testing.T) {
	if got := collect("int a{}"); !slices.Equal(got, []string{"int", " ", "a", "{", "}"}) {
		t.Errorf("unexpected tokens: %v", got)
	}
}

func TestUnit_GenerateTokens_DoubleQuotedString(t *testing.T) {
	cases := []struct {
		src  string
		want []string
	}{
		{`""`, []string{`""`}},
		{`"x\"xx")`, []string{`"x\"xx"`, ")"}},
	}
	for _, c := range cases {
		if got := collect(c.src); !slices.Equal(got, c.want) {
			t.Errorf("collect(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestUnit_GenerateTokens_SingleQuotedString(t *testing.T) {
	// '\'' → one token (escaped quote inside)
	if got := collect("'\\''"); !slices.Equal(got, []string{"'\\''"}) {
		t.Errorf("collect(`'\\''`) = %v", got)
	}
	// '\\\'  → 5 tokens: each char separately (no valid closing quote found before end)
	if got := collect(`'\\\` + `'`); !slices.Equal(got, []string{"'", `\`, `\`, `\`, "'"}) {
		t.Errorf("collect(`'\\\\\\' `) = %v", got)
	}
}

func TestUnit_GenerateTokens_MultiLineString(t *testing.T) {
	toks := collect("\"sss\nsss\" t")
	found := false
	for _, tok := range toks {
		if tok == "t" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 't' in tokens, got %v", toks)
	}
}

func TestUnit_GenerateTokens_LineNumber2(t *testing.T) {
	toks := collect("abc\ndef")
	found := false
	for _, tok := range toks {
		if tok == "def" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'def' in tokens")
	}
}

// --- Macro tests ---

func TestUnit_GenerateTokens_MacroDefine(t *testing.T) {
	define := "#define xx()\\\n                       abc"
	src := define + "\n                    int"
	toks := collect(src)
	want := []string{define, "\n", "                    ", "int"}
	if !slices.Equal(toks, want) {
		t.Errorf("got %v, want %v", toks, want)
	}
}

func TestUnit_GenerateTokens_MacroInclude(t *testing.T) {
	toks := collect(`#include "abc"`)
	if !slices.Equal(toks, []string{`#include "abc"`}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_MacroIf(t *testing.T) {
	toks := collect("#if abc\n")
	if !slices.Equal(toks, []string{"#if abc", "\n"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_MacroIfdef(t *testing.T) {
	toks := collect("#ifdef abc\n")
	if !slices.Equal(toks, []string{"#ifdef abc", "\n"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_MacroLineContinuer(t *testing.T) {
	toks := collect("#define a \\\nb\n t")
	found := false
	for _, tok := range toks {
		if tok == "t" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 't' in tokens, got %v", toks)
	}
}

func TestUnit_GenerateTokens_MacroComplex(t *testing.T) {
	src := " # define yyMakeArray(ptr, count, size)     { MakeArray (ptr, count, size); \\\n" +
		"                   yyCheckMemory (* ptr); }\n" +
		"                   t\n" +
		"                "
	toks := collect(src)
	found := false
	for _, tok := range toks {
		if tok == "t" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 't' in tokens, got %v", toks)
	}
}

func TestUnit_GenerateTokens_HalfCommentAfterMacro(t *testing.T) {
	// #define A/*\n*/ → 2 tokens: the macro and the block comment
	toks := collect("#define A/*\n*/")
	if len(toks) != 2 {
		t.Errorf("expected 2 tokens, got %d: %v", len(toks), toks)
	}
}

func TestUnit_GenerateTokens_BlockCommentInDefine(t *testing.T) {
	// #define A \\\n/*\\\n*/ → 1 token (continuation keeps it together)
	toks := collect("#define A \\\n/*\\\n*/")
	if len(toks) != 1 {
		t.Errorf("expected 1 token, got %d: %v", len(toks), toks)
	}
}

// --- Comment tests ---

func TestUnit_GenerateTokens_CStyleComment(t *testing.T) {
	toks := collect("/***\n**/")
	if !slices.Equal(toks, []string{"/***\n**/"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_CppStyleComment(t *testing.T) {
	toks := collect("//aaa\n")
	if !slices.Equal(toks, []string{"//aaa", "\n"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_CppCommentMultiLine(t *testing.T) {
	// //a\<newline>b — line continuation inside comment keeps it one token
	toks := collect("//a\\\nb")
	if !slices.Equal(toks, []string{"//a\\\nb"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_CommentedComment(t *testing.T) {
	toks := collect(" /*/*/")
	if !slices.Equal(toks, []string{" ", "/*/*/"}) {
		t.Errorf("unexpected: %v", toks)
	}
}

func TestUnit_GenerateTokens_CppCommentWithCode(t *testing.T) {
	toks := collect("//abc\n t")
	found := false
	for _, tok := range toks {
		if tok == "t" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 't', got %v", toks)
	}
}

func TestUnit_GenerateTokens_CStyleCommentWithCode(t *testing.T) {
	toks := collect("/*abc\n*/ t")
	found := false
	for _, tok := range toks {
		if tok == "t" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 't', got %v", toks)
	}
}

func TestUnit_GenerateTokens_CStyleCommentWithBackslash(t *testing.T) {
	comment := "/**a/*/"
	toks := collect(comment)
	if !slices.Equal(toks, []string{comment}) {
		t.Errorf("unexpected: %v", toks)
	}
}
