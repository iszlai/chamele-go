// Package clike implements the C/C++ language reader, shared as a base by
// Java, C#, Kotlin, TypeScript, and several other readers.
package clike

import (
	"iter"
	"regexp"
	"strings"

	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
)

func init() {
	languages.Register(NewCLikeReader())
}

// CLikeReader is the language reader for C and C++.
type CLikeReader struct {
	tokenizer.BaseReader
}

// NewCLikeReader returns a CLikeReader registered for C/C++ extensions.
func NewCLikeReader() *CLikeReader {
	return &CLikeReader{BaseReader: tokenizer.NewBaseReader()}
}

func (r *CLikeReader) Extensions() []string    { return []string{"c", "cpp", "cc", "cxx", "h", "hpp"} }
func (r *CLikeReader) LanguageNames() []string { return []string{"cpp", "c"} }

// clikeAddition adds C++ raw-string and floating-point literal patterns to the base tokenizer.
const clikeAddition = `` +
	`|R"[^(\\]*\((?:[^)]|\)[^"])*\)"` +
	`|(?:\d*\.\d+(?:[eE][-+]?\d+)?)` +
	`|(?:\d+\.(?:\d+)?(?:[eE][-+]?\d+)?)`

func (r *CLikeReader) Tokenize(src []byte) iter.Seq[string] {
	return tokenizer.GenerateTokens(src, clikeAddition)
}

var macroRe = regexp.MustCompile(`(?ms)#\s*(\w+)\s*(.*)`)

// Preprocess handles ~Foo destructor joining, whitespace stripping, and
// C preprocessor directives (#if/#elif → CCN +1, #include → two tokens).
func (r *CLikeReader) Preprocess(tokens iter.Seq[string], ctx languages.Context) iter.Seq[string] {
	return func(yield func(string) bool) {
		tilde := false
		for tok := range tokens {
			if tok == "~" {
				tilde = true
				continue
			}
			if tilde {
				tilde = false
				if !yield("~" + tok) {
					return
				}
				continue
			}
			if isHSpace(tok) {
				continue
			}
			if tok == "\n" {
				if !yield(tok) {
					return
				}
				continue
			}
			if m := macroRe.FindStringSubmatch(tok); m != nil {
				directive, rest := m[1], m[2]
				switch directive {
				case "if", "ifdef", "elif":
					ctx.AddCondition(1)
				case "include":
					if !yield("#include") {
						return
					}
					arg := strings.TrimSpace(rest)
					if arg == "" {
						arg = `""`
					}
					if !yield(arg) {
						return
					}
				}
				for range strings.Split(rest, "\n")[1:] {
					if !yield("\n") {
						return
					}
				}
				continue
			}
			if !yield(tok) {
				return
			}
		}
	}
}

// GetComment returns the comment body if tok is a C/C++ comment.
func (r *CLikeReader) GetComment(tok string) (string, bool) {
	if len(tok) >= 2 && (tok[:2] == "//" || tok[:2] == "/*") {
		return tok[2:], true
	}
	return "", false
}

// GetConditions returns the C-like CCN condition token set.
func (r *CLikeReader) GetConditions() map[string]struct{} {
	return map[string]struct{}{
		"if": {}, "for": {}, "while": {}, "catch": {},
		"&&": {}, "||": {}, "case": {}, "?": {},
	}
}

// RunTokens feeds processed tokens to the three parallel state machines.
func (r *CLikeReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	RunParallel(tokens, ctx, NewCLikeStates, NewCLikeNestingStackStates, NewCppRValueRefStates)
}

// RunParallel runs a set of state machine constructors over a token stream.
// Language readers that extend CLikeReader override which machines they use.
func RunParallel(
	tokens iter.Seq[string],
	ctx languages.Context,
	constructors ...func(languages.Context) *tokenizer.Machine,
) {
	machines := make([]*tokenizer.Machine, len(constructors))
	for i, ctor := range constructors {
		machines[i] = ctor(ctx)
	}
	for tok := range tokens {
		for _, m := range machines {
			m.Call(tok)
		}
	}
	for _, m := range machines {
		m.StatemachineBeforeReturn()
	}
}

// ---- CppRValueRefStates ----

type cppRValueRefStates struct {
	m   *tokenizer.Machine
	ctx languages.Context
}

// NewCppRValueRefStates returns the machine that corrects CCN for && used as
// r-value references (not logical-and) and inside typedef declarations.
func NewCppRValueRefStates(ctx languages.Context) *tokenizer.Machine {
	s := &cppRValueRefStates{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *cppRValueRefStates) stateGlobal(tok string) bool {
	switch tok {
	case "&&":
		s.m.Next(tokenizer.ReadUntilThen([]string{"=", ";", "{", "}", ")"}, func(stop string, _ []string) {
			if stop == "=" {
				s.ctx.AddCondition(-1)
			}
			s.m.Next(s.stateGlobal)
		}))
	case "typedef":
		s.m.Next(tokenizer.ReadUntilThen([]string{";"}, func(_ string, collected []string) {
			count := 0
			for _, t := range collected {
				if t == "&&" {
					count++
				}
			}
			s.ctx.AddCondition(-count)
			s.m.Next(s.stateGlobal)
		}))
	}
	return false
}

// ---- CLikeNestingStackStates ----

var namespaceSeparators = map[string]bool{
	"<": true, ":": true, "final": true, "[": true, "extends": true, "implements": true,
}

type clikeNestingStackStates struct {
	m   *tokenizer.Machine
	ctx languages.Context
}

// NewCLikeNestingStackStates returns the machine that tracks {}/{} nesting and
// class/namespace declarations.
func NewCLikeNestingStackStates(ctx languages.Context) *tokenizer.Machine {
	s := &clikeNestingStackStates{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

func (s *clikeNestingStackStates) stateGlobal(tok string) bool {
	switch tok {
	case "template":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateGlobal, func(_ string) {}))
	case ".":
		s.m.Next(func(_ string) bool { s.m.Next(s.stateGlobal); return false })
	case "struct", "class", "namespace", "union":
		s.m.Next(s.stateReadNamespace)
	case "{":
		s.ctx.AddBareNesting()
	case "}":
		s.ctx.PopNesting()
	}
	return false
}

func (s *clikeNestingStackStates) stateReadNamespace(tok string) bool {
	if tok == "[" {
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateReadNamespace, func(_ string) {}), tok)
	} else {
		s.m.Next(s.stateReadNamespaceName, tok)
	}
	return false
}

func (s *clikeNestingStackStates) stateReadNamespaceName(tok string) bool {
	s.m.Next(tokenizer.ReadUntilThen([]string{"(", "{", ";"}, func(stop string, saved []string) {
		s.m.Next(s.stateGlobal)
		if stop == "{" {
			var name strings.Builder
			for _, t := range saved {
				if namespaceSeparators[t] {
					break
				}
				name.WriteString(t)
			}
			s.ctx.AddNamespace(name.String())
		}
	}), tok)
	return false
}

// ---- CLikeStates ----

// CLikeStates is the function-detection state machine for C/C++.
// It is exported so that JavaStates and other derived readers can compose it.
type CLikeStates struct {
	m            *tokenizer.Machine
	ctx          languages.Context
	bracketStack []string
	savedTokens  []string
}

// NewCLikeStates returns the function-detection state machine.
func NewCLikeStates(ctx languages.Context) *tokenizer.Machine {
	s := &CLikeStates{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

// NewExportedCLikeStates returns a CLikeStates value for use by language readers
// that need to compose CLike state machine logic (e.g. Java).
// Call M().SetInitialState(yourStateGlobal) after this to override the initial state.
func NewExportedCLikeStates(ctx languages.Context) CLikeStates {
	s := CLikeStates{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s
}

// M returns the underlying machine.
func (s *CLikeStates) M() *tokenizer.Machine { return s.m }

// Ctx returns the analysis context.
func (s *CLikeStates) Ctx() languages.Context { return s.ctx }

// StateGlobal is the exported entry point to the C-like global state.
// Derived readers call this to delegate tokens back to CLike's default handling.
func (s *CLikeStates) StateGlobal(tok string) { s.stateGlobal(tok) }

// TryNewFunction attempts to start tracking a new function named name.
func (s *CLikeStates) TryNewFunction(name string) {
	s.ctx.TryNewFunction(name)
	s.m.Next(s.stateFunction)
	if name == "operator" {
		s.m.Next(s.stateOperator)
	}
}

func (s *CLikeStates) stateGlobal(tok string) bool {
	if len(tok) > 0 && (isAlpha(tok[0]) || tok[0] == '_' || tok[0] == '~') {
		s.TryNewFunction(tok)
	} else if tok == "[" {
		s.m.Next(s.stateLambdaCheck)
	}
	return false
}

func (s *CLikeStates) stateFunction(tok string) bool {
	switch tok {
	case "(":
		s.bracketStack = nil
		s.m.Next(s.stateDecFn(), tok)
	case "::":
		s.ctx.AddToFunctionName(tok)
		s.m.Next(s.stateNameWithSpace)
	case "<":
		s.m.Next(s.stateTemplateInNameFn(), tok)
	default:
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *CLikeStates) stateTemplateInNameFn() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateFunction, func(tok string) {
		s.ctx.AddToFunctionName(tok)
	})
}

func (s *CLikeStates) stateOperator(tok string) bool {
	if tok != "(" {
		s.m.Next(s.stateOperatorNext)
	}
	s.ctx.AddToFunctionName(" " + tok)
	return false
}

func (s *CLikeStates) stateOperatorNext(tok string) bool {
	if tok == "(" {
		s.stateFunction(tok)
	} else {
		s.ctx.AddToFunctionName(" " + tok)
	}
	return false
}

func (s *CLikeStates) stateNameWithSpace(tok string) bool {
	if tok == "operator" {
		s.m.Next(s.stateOperator)
	} else {
		s.m.Next(s.stateFunction)
	}
	s.ctx.AddToFunctionName(tok)
	return false
}

func (s *CLikeStates) stateDecFn() tokenizer.StateFn {
	return tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateDecToImp, func(tok string) {
		switch {
		case tok == "(" || tok == "<":
			s.bracketStack = append(s.bracketStack, tok)
		case tok == ")" || tok == ">":
			if len(s.bracketStack) > 0 {
				s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			} else {
				s.m.Next(s.stateGlobal)
			}
		case len(s.bracketStack) == 1:
			if tok != "void" {
				s.ctx.Parameter(tok)
			}
			return
		}
		s.ctx.AddToLongFunctionName(tok)
	})
}

func (s *CLikeStates) stateDecToImp(tok string) bool {
	switch tok {
	case "const", "&", "&&":
		s.ctx.AddToLongFunctionName(" " + tok)
	case "throw":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateDecToImp, func(_ string) {}))
	case "throws":
		s.m.Next(tokenizer.ReadUntilThen([]string{";", "{"}, func(stop string, _ []string) {
			s.m.Next(s.stateDecToImp)
			s.stateDecToImp(stop)
		}))
	case "->":
		s.m.Next(tokenizer.ReadUntilThen([]string{";", "{"}, func(stop string, _ []string) {
			s.m.Next(s.stateDecToImp)
			s.stateDecToImp(stop)
		}))
	case "noexcept":
		s.m.Next(s.stateNoexcept)
	case "(":
		longName := s.ctx.CurrentFunctionLongName()
		s.TryNewFunction(longName)
		s.stateFunction(tok)
	case "{":
		s.m.Next(s.stateEnteringImp, "{")
	case ":":
		s.m.Next(s.stateInitializationList)
	case "[":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateDecToImp, func(_ string) {}), tok)
	default:
		if len(tok) > 0 && !isAlpha(tok[0]) && tok[0] != '_' {
			s.m.Next(s.stateGlobal, tok)
		} else {
			s.m.Next(s.stateOldCParams)
			s.savedTokens = []string{tok}
		}
	}
	return false
}

func (s *CLikeStates) stateNoexcept(tok string) bool {
	if tok == "(" {
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateDecToImp, func(_ string) {}))
	} else {
		s.m.Next(s.stateDecToImp, tok)
	}
	return false
}

func (s *CLikeStates) stateOldCParams(tok string) bool {
	s.savedTokens = append(s.savedTokens, tok)
	switch tok {
	case ";":
		s.savedTokens = nil
		s.m.Next(s.stateDecToImp)
	case "{":
		if len(s.savedTokens) == 2 {
			s.savedTokens = nil
			s.stateDecToImp(tok)
			return false
		}
		s.m.Next(s.stateGlobal)
		for _, t := range s.savedTokens {
			s.stateGlobal(t)
		}
	case "(":
		s.m.Next(s.stateGlobal)
		for _, t := range s.savedTokens {
			s.stateGlobal(t)
		}
	}
	return false
}

func (s *CLikeStates) stateEnteringImp(tok string) bool {
	s.ctx.ConfirmNewFunction()
	s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "{", "}", s.stateGlobal, func(_ string) {}), tok)
	return false
}

func (s *CLikeStates) stateInitializationList(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateEnteringImp, "{")
		return false
	}
	s.m.Next(tokenizer.ReadUntilThen([]string{"(", "{"}, func(stop string, _ []string) {
		if stop == "(" {
			s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateInitializationList, func(_ string) {}), stop)
		} else {
			s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "{", "}", s.stateInitializationList, func(_ string) {}), stop)
		}
	}), tok)
	return false
}

// Lambda support

func (s *CLikeStates) stateLambdaCheck(tok string) bool {
	switch tok {
	case "]":
		s.m.Next(s.stateLambdaParams)
	case "[":
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "[", "]", s.stateDecToImp, func(_ string) {}))
	default:
		s.m.Next(s.stateLambdaCapture)
	}
	return false
}

func (s *CLikeStates) stateLambdaCapture(tok string) bool {
	if tok == "]" {
		s.m.Next(s.stateLambdaParams)
	}
	return false
}

func (s *CLikeStates) stateLambdaParams(tok string) bool {
	if tok == "(" {
		s.bracketStack = []string{"("}
		s.m.Next(s.stateLambdaParamList)
	} else {
		s.m.Next(s.stateLambdaBody)
		s.stateLambdaBody(tok)
	}
	return false
}

func (s *CLikeStates) stateLambdaParamList(tok string) bool {
	switch tok {
	case "(", "<", "[":
		s.bracketStack = append(s.bracketStack, tok)
	case ")":
		if len(s.bracketStack) > 0 && s.bracketStack[len(s.bracketStack)-1] == "(" {
			s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			if len(s.bracketStack) == 0 {
				s.m.Next(s.stateLambdaBody)
			}
		}
	case ">":
		if len(s.bracketStack) > 0 && s.bracketStack[len(s.bracketStack)-1] == "<" {
			s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
		}
	case "]":
		if len(s.bracketStack) > 0 && s.bracketStack[len(s.bracketStack)-1] == "[" {
			s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
		}
	}
	return false
}

func (s *CLikeStates) stateLambdaBody(tok string) bool {
	switch tok {
	case "{":
		s.bracketStack = []string{"{"}
		s.m.Next(s.stateLambdaBodySkip)
	case ";", ",", ")":
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

func (s *CLikeStates) stateLambdaBodySkip(tok string) bool {
	switch tok {
	case "{":
		s.bracketStack = append(s.bracketStack, "{")
	case "}":
		if len(s.bracketStack) > 0 && s.bracketStack[len(s.bracketStack)-1] == "{" {
			s.bracketStack = s.bracketStack[:len(s.bracketStack)-1]
			if len(s.bracketStack) == 0 {
				s.m.Next(s.stateGlobal)
			}
		}
	}
	return false
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isHSpace(s string) bool {
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\r' {
			return false
		}
	}
	return len(s) > 0
}
