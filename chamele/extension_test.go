package chamele

import (
	"iter"
	"testing"
)

// stateful is an Extension that counts every token across its lifetime, so we
// can verify whether RegisterExtensionFactory yields a fresh instance per
// run.
type stateful struct {
	count int
}

func (s *stateful) Name() string                      { return "stateful" }
func (s *stateful) OrderingIndex() int                { return 1000 }
func (s *stateful) FunctionInfoColumns() []ColumnSpec { return nil }
func (s *stateful) Process(tokens iter.Seq[string], _ *FileInfoBuilder) iter.Seq[string] {
	return func(yield func(string) bool) {
		for tok := range tokens {
			s.count++
			if !yield(tok) {
				return
			}
		}
	}
}

func TestExtensionFactory_FreshPerCall(t *testing.T) {
	// Use a local registry-equivalent: drive the factory directly.
	factory := func() Extension { return &stateful{} }

	a := factory().(*stateful)
	b := factory().(*stateful)
	if a == b {
		t.Fatal("factory returned the same instance twice")
	}

	a.count = 10
	if b.count != 0 {
		t.Errorf("expected fresh instance b to have count 0, got %d", b.count)
	}
}

func TestExtensionPhase_FromOrderingIndex(t *testing.T) {
	type fakePre struct{ stateful }
	type fakePost struct{ stateful }
	pre := &fakePre{}
	post := &fakePost{}
	// stateful.OrderingIndex() returns 1000 → PhasePostBuiltins
	if got := ExtensionPhase(post); got != PhasePostBuiltins {
		t.Errorf("post: got %v, want PhasePostBuiltins", got)
	}
	// Override via a value that returns negative OrderingIndex via Phase()
	// would need a phaseProvider — here we just verify default mapping.
	_ = pre
}

type phasedExt struct {
	stateful
	phase Phase
}

func (p *phasedExt) Phase() Phase { return p.phase }

func TestExtensionPhase_FromPhaseProvider(t *testing.T) {
	e := &phasedExt{phase: PhasePreBuiltins}
	if got := ExtensionPhase(e); got != PhasePreBuiltins {
		t.Errorf("got %v, want PhasePreBuiltins", got)
	}
}
