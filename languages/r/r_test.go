package r_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/r"
)

func rFunctions(src string) []*chamele.FunctionInfo {
	rd := r.NewRReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.r", []byte(src), rd).Functions
}

func TestUnit_R_Empty(t *testing.T) {
	if got := rFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_R_SimpleFunction(t *testing.T) {
	got := rFunctions(`
simple_func <- function(x, y = 5) {
    result <- x + y
    return(result)
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "simple_func" {
		t.Errorf("name = %q, want simple_func", got[0].Name)
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_R_AlternativeAssignment(t *testing.T) {
	got := rFunctions(`
simple_func2 = function(a, b) {
    a + b -> result
    result
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "simple_func2" {
		t.Errorf("name = %q, want simple_func2", got[0].Name)
	}
}

func TestUnit_R_IfElseComplexity(t *testing.T) {
	got := rFunctions(`
complex_control <- function(x) {
    if (x > 0) {
        print("positive")
    } else if (x < 0) {
        print("negative")
    } else {
        print("zero")
    }
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// base(1) + if(1) + else if(1) = 3
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_R_NestedLoops(t *testing.T) {
	got := rFunctions(`
nested_loops <- function(n) {
    for (i in 1:n) {
        for (j in 1:i) {
            if (i %% j == 0) {
                print(paste(i, j))
            }
        }
    }
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// base(1) + for(1) + for(1) + if(1) = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_R_WhileRepeat(t *testing.T) {
	got := rFunctions(`
iterative_func <- function(x) {
    while (x > 1) {
        x <- x / 2
    }
    repeat {
        x <- x * 2
        if (x > 100) break
    }
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// base(1) + while(1) + repeat(1) + if(1) = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_R_SingleLineFunction(t *testing.T) {
	got := rFunctions(`
one_liner <- function(x) x + 1
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "one_liner" {
		t.Errorf("name = %q, want one_liner", got[0].Name)
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_R_EmptyFunction(t *testing.T) {
	got := rFunctions(`
empty_func <- function() {}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "empty_func" {
		t.Errorf("name = %q, want empty_func", got[0].Name)
	}
}

func TestUnit_R_AnonymousFunction(t *testing.T) {
	got := rFunctions(`
apply_func <- function(data) {
    sapply(data, function(x) {
        if (x > 0) {
            return(x^2)
        } else {
            return(0)
        }
    })
}
`)
	if len(got) < 1 {
		t.Fatalf("expected at least 1, got %d", len(got))
	}
}

func TestUnit_R_Switch(t *testing.T) {
	got := rFunctions(`
switch_func <- function(type) {
    switch(type,
        "a" = 1,
        "b" = 2,
        "c" = 3,
        default = 0
    )
}
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "switch_func" {
		t.Errorf("name = %q, want switch_func", got[0].Name)
	}
	// base(1) + switch(1) = 2
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}
