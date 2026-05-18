// Package java implements the Java language reader.
package java

import (
	"iter"

	"github.com/iszlai/chamele-go/internal/stringx"
	"github.com/iszlai/chamele-go/internal/tokenizer"
	"github.com/iszlai/chamele-go/languages"
	"github.com/iszlai/chamele-go/languages/clike"
)

func init() {
	languages.Register(NewJavaReader())
}

// JavaReader handles Java source files. It delegates tokenization and
// comment handling to CLikeReader and uses a Java-specific state machine
// for function detection.
type JavaReader struct {
	*clike.CLikeReader
}

func NewJavaReader() *JavaReader {
	return &JavaReader{CLikeReader: clike.NewCLikeReader()}
}

func (r *JavaReader) Extensions() []string    { return []string{"java"} }
func (r *JavaReader) LanguageNames() []string { return []string{"java"} }

func (r *JavaReader) Tokenize(src []byte) iter.Seq[string] {
	return r.CLikeReader.Tokenize(src)
}

// RunTokens drives two parallel machines: a Java-specific function detector
// and the shared C-like nesting tracker that manages namespace qualification.
func (r *JavaReader) RunTokens(tokens iter.Seq[string], ctx languages.Context) {
	clike.RunParallel(tokens, ctx,
		newJavaMachine,
		clike.NewCLikeNestingStackStates,
	)
}

// ---- Java state machine ----
//
// Design note: class body `{` is NOT tracked with ReadInsideBracketsThen
// because function detection changes m.state (abandoning any outer closure).
// Instead we rely on CLikeNestingStackStates to push/pop namespace items on
// the nesting stack, so ctx.WithNamespace automatically qualifies method names
// as "ClassName::method" without the Java machine needing to track class names.

type javaMachine struct {
	m            *tokenizer.Machine
	ctx          languages.Context
	bracketStack []string
}

func newJavaMachine(ctx languages.Context) *tokenizer.Machine {
	s := &javaMachine{ctx: ctx}
	s.m = tokenizer.NewMachine()
	s.m.SetInitialState(s.stateGlobal)
	return s.m
}

// stateGlobal is the top-level state. Any alpha token is tried as a potential
// function name (just as in CLikeStates); non-function tokens are discarded
// when no `(` follows.
func (s *javaMachine) stateGlobal(tok string) bool {
	switch {
	case tok == "@":
		s.m.Next(s.stateAnnotation)
	case tok == "class" || tok == "enum" || tok == "record":
		// Skip the class/enum/record header and enter the body via stateClassHeader.
		// CLikeNestingStackStates handles the namespace push when { is seen.
		s.m.Next(s.stateClassHeader)
	case len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_'):
		// ctx.TryNewFunction uses ctx.WithNamespace which incorporates any
		// class namespace managed by CLikeNestingStackStates.
		s.ctx.TryNewFunction(tok)
		s.m.Next(s.stateFunction)
	}
	return false
}

// stateAnnotation: consume the annotation name, then transition.
func (s *javaMachine) stateAnnotation(tok string) bool {
	if len(tok) > 0 && (stringx.IsAlpha(tok[0]) || tok[0] == '_') {
		s.m.Next(s.statePostAnnotation)
		return false
	}
	s.m.Next(s.stateGlobal, tok)
	return false
}

func (s *javaMachine) statePostAnnotation(tok string) bool {
	switch tok {
	case ".":
		s.m.Next(s.stateAnnotation) // @pkg.Annotation — skip next segment
	case "(":
		// @Annotation(args) — skip the argument list
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "(", ")", s.stateGlobal, func(_ string) {}), tok)
	default:
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

// stateClassHeader: skip class/enum/record header tokens until { is seen.
// The { transitions back to stateGlobal; CLikeNestingStackStates adds the
// class namespace at that point.
func (s *javaMachine) stateClassHeader(tok string) bool {
	if tok == "{" {
		s.m.Next(s.stateGlobal)
	}
	return false
}

// stateFunction: we have a candidate function name; look for ( to confirm.
func (s *javaMachine) stateFunction(tok string) bool {
	switch tok {
	case "(":
		s.bracketStack = nil
		s.m.Next(s.stateDecFn(), tok)
	case "<":
		// Generic type in the function name (e.g. <T extends Foo>) — skip contents.
		s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "<", ">", s.stateFunction, func(t string) {
			s.ctx.AddToFunctionName(t)
		}), tok)
	default:
		// Not a function — discard the candidate and re-process tok.
		s.m.Next(s.stateGlobal, tok)
	}
	return false
}

// stateDecFn tracks the parameter list ( … ).
func (s *javaMachine) stateDecFn() tokenizer.StateFn {
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

// stateDecToImp: after parameter list, waiting for { (confirm), throws (skip), or ; (discard).
func (s *javaMachine) stateDecToImp(tok string) bool {
	switch tok {
	case "{":
		s.m.Next(s.stateEnteringImp, "{")
	case "throws":
		s.m.Next(tokenizer.ReadUntilThen([]string{";", "{"}, func(stop string, _ []string) {
			s.m.Next(s.stateDecToImp)
			s.stateDecToImp(stop)
		}))
	case ";":
		// Abstract or interface method — no body, discard candidate.
		s.m.Next(s.stateGlobal)
	}
	return false
}

func (s *javaMachine) stateEnteringImp(tok string) bool {
	s.ctx.ConfirmNewFunction()
	s.m.Next(tokenizer.ReadInsideBracketsThen(s.m, "{", "}", s.stateGlobal, func(_ string) {}), tok)
	return false
}
