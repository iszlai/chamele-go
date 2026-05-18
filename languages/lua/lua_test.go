package lua_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/lua"
)

func luaFunctions(src string) []*chamele.FunctionInfo {
	r := lua.NewLuaReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.lua", []byte(src), r).Functions
}

func TestUnit_Lua_Empty(t *testing.T) {
	if got := luaFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Lua_NoFunction(t *testing.T) {
	if got := luaFunctions(` p "1" `); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Lua_OneFunction(t *testing.T) {
	got := luaFunctions(`
function f
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "f" {
		t.Errorf("name = %q, want f", got[0].Name)
	}
	if got[0].ParameterCount() != 0 {
		t.Errorf("params = %d, want 0", got[0].ParameterCount())
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Lua_TwoFunctions(t *testing.T) {
	got := luaFunctions(`
function f
end
function g
end
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[1].Name != "g" {
		t.Errorf("got[1].Name = %q, want g", got[1].Name)
	}
}

func TestUnit_Lua_DoBlock(t *testing.T) {
	got := luaFunctions(`
function k
end
do
    function g
    end
end
function f
end
`)
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
}

func TestUnit_Lua_ForAndWhile(t *testing.T) {
	got := luaFunctions(`
function factorial(n)
  local x = 1
  for i = 2, n do
    x = x * i
  end
  while a do
    a=a-1
  end
  return x
end
function g
end
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("factorial CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Lua_RepeatUntil(t *testing.T) {
	got := luaFunctions(`
function f(n)
    repeat
      --statements
    until condition
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Lua_If(t *testing.T) {
	got := luaFunctions(`
function a(n)
    if a then
    elseif b then
    else
    end
end
function a(n)
end
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Lua_ClassMethod(t *testing.T) {
	got := luaFunctions(`
function V.f(n)
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "V.f" {
		t.Errorf("name = %q, want V.f", got[0].Name)
	}
}

func TestUnit_Lua_Anonymous(t *testing.T) {
	got := luaFunctions(`
function(self, neigh, id)
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "(anonymous)" {
		t.Errorf("name = %q, want (anonymous)", got[0].Name)
	}
}

func TestUnit_Lua_AnonymousWithAssignment(t *testing.T) {
	got := luaFunctions(`
a = function(self, neigh, id)
end
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "a" {
		t.Errorf("name = %q, want a", got[0].Name)
	}
}

func TestUnit_Lua_NestedFunctions(t *testing.T) {
	got := luaFunctions(`
function addn(x)
  function sum(y)
    return x+y
  end
  return sum
end
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "sum" {
		t.Errorf("got[0].Name = %q, want sum", got[0].Name)
	}
	if got[1].Name != "addn" {
		t.Errorf("got[1].Name = %q, want addn", got[1].Name)
	}
}
