package chamele

import "testing"

func TestUnit_FunctionInfo_ParameterCount(t *testing.T) {
	cases := []struct {
		name   string
		tokens []string // tokens fed to AddParameter
		want   int
	}{
		{
			name:   "empty",
			tokens: nil,
			want:   0,
		},
		{
			name:   "c_style_two_params",
			tokens: []string{"int", "a", ",", "char", "*", "b"},
			want:   2,
		},
		{
			name:   "python_annotated_with_default",
			tokens: []string{"a", ":", "int", "=", "5", ",", "b", ":", "str"},
			want:   2,
		},
		{
			name:   "trailing_comma",
			tokens: []string{"a", ",", "b", ","},
			want:   2,
		},
		{
			name:   "single_word",
			tokens: []string{"x"},
			want:   1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := NewFunctionInfo("f", "test.go", 1)
			for _, tok := range c.tokens {
				f.AddParameter(tok)
			}
			if got := f.ParameterCount(); got != c.want {
				t.Errorf("ParameterCount() = %d, want %d (full_params=%v)",
					got, c.want, f.FullParameters)
			}
		})
	}
}

func TestUnit_FunctionInfo_Location(t *testing.T) {
	f := NewFunctionInfo("foo", "main.go", 10)
	f.EndLine = 20
	want := " foo@10-20@main.go"
	if got := f.Location(); got != want {
		t.Errorf("Location() = %q, want %q", got, want)
	}
}

func TestUnit_FunctionInfo_UnqualifiedName(t *testing.T) {
	cases := []struct{ name, want string }{
		{"foo", "foo"},
		{"Foo::bar", "bar"},
		{"A::B::C", "C"},
	}
	for _, c := range cases {
		f := NewFunctionInfo(c.name, "", 0)
		if got := f.UnqualifiedName(); got != c.want {
			t.Errorf("UnqualifiedName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestUnit_FunctionInfo_AddToLongName_Space(t *testing.T) {
	f := NewFunctionInfo("foo", "", 0)
	f.AddToLongName("bar") // both alpha → space inserted
	if f.LongName != "foo bar" {
		t.Errorf("LongName = %q, want %q", f.LongName, "foo bar")
	}
}

func TestUnit_FunctionInfo_AddToLongName_NoSpace(t *testing.T) {
	f := NewFunctionInfo("foo", "", 0)
	f.AddToLongName("(") // ( is not alpha → no space
	if f.LongName != "foo(" {
		t.Errorf("LongName = %q, want %q", f.LongName, "foo(")
	}
}
