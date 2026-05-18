package fortran_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/fortran"
)

func fortranFunctions(src string) []*chamele.FunctionInfo {
	r := fortran.NewFortranReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.f90", []byte(src), r).Functions
}

func TestUnit_Fortran_Empty(t *testing.T) {
	if got := fortranFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_Fortran_Subroutine(t *testing.T) {
	got := fortranFunctions(`
SUBROUTINE test(a, b)
    REAL :: a
    REAL :: b
END SUBROUTINE test
subroutine test2
endsubroutine test2
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "test" {
		t.Errorf("got[0].Name = %q, want test", got[0].Name)
	}
	if got[1].Name != "test2" {
		t.Errorf("got[1].Name = %q, want test2", got[1].Name)
	}
}

func TestUnit_Fortran_Function(t *testing.T) {
	got := fortranFunctions(`
FUNCTION test(a, b)
    REAL :: a
    REAL :: b
END FUNCTION test
function test2
endfunction test2
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "test" {
		t.Errorf("got[0].Name = %q, want test", got[0].Name)
	}
}

func TestUnit_Fortran_Module(t *testing.T) {
	got := fortranFunctions(`
module test
    subroutine test2
    endsubroutine test2
end module test
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "test::test2" {
		t.Errorf("name = %q, want test::test2", got[0].Name)
	}
}

func TestUnit_Fortran_CCN_If(t *testing.T) {
	got := fortranFunctions(`
subroutine test
    if (a) call sub(a)
    if (b) then
        call sub(b)
    end if
endsubroutine test
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Fortran_CCN_IfElse(t *testing.T) {
	got := fortranFunctions(`
subroutine test
    if (a) then
        call sub(a)
    else
        call sub(-a)
    end if
    if (b) then
        call sub(b)
    else  if (c) then
        call sub(c)
    end if
endsubroutine test
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 3 {
		t.Errorf("CCN = %d, want 3", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Fortran_CCN_Complex(t *testing.T) {
	got := fortranFunctions(`
subroutine test
    if (a .AND. b) then
        do b = 1, 10
            select case (b)
            case (1)
                do xxx
                    do xxx
                        call sub()
                    end do
                enddo
            case (2)
                call sub()
            endselect
        end do
    else if (a .OR. b) then
        sub()
    endif
endsubroutine test
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 10 {
		t.Errorf("CCN = %d, want 10", got[0].CyclomaticComplexity)
	}
}
