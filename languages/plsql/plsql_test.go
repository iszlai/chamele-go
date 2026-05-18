package plsql_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/plsql"
)

func plsqlFunctions(src string) []*chamele.FunctionInfo {
	r := plsql.NewPLSQLReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.sql", []byte(src), r).Functions
}

func TestUnit_PLSQL_Empty(t *testing.T) {
	if got := plsqlFunctions(""); len(got) != 0 {
		t.Errorf("expected 0, got %d", len(got))
	}
}

func TestUnit_PLSQL_SimpleProcedure(t *testing.T) {
	got := plsqlFunctions(`
CREATE OR REPLACE PROCEDURE simple_proc IS
BEGIN
    NULL;
END simple_proc;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "simple_proc" {
		t.Errorf("name = %q, want simple_proc", got[0].Name)
	}
	if got[0].CyclomaticComplexity != 1 {
		t.Errorf("CCN = %d, want 1", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_SimpleFunction(t *testing.T) {
	got := plsqlFunctions(`
CREATE OR REPLACE FUNCTION simple_func RETURN NUMBER IS
BEGIN
    RETURN 1;
END simple_func;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "simple_func" {
		t.Errorf("name = %q, want simple_func", got[0].Name)
	}
}

func TestUnit_PLSQL_Parameters(t *testing.T) {
	got := plsqlFunctions(`
CREATE OR REPLACE PROCEDURE process_order(
    p_order_id IN NUMBER,
    p_status OUT VARCHAR2
) IS
BEGIN
    NULL;
END process_order;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].ParameterCount() != 2 {
		t.Errorf("params = %d, want 2", got[0].ParameterCount())
	}
}

func TestUnit_PLSQL_CCN_If(t *testing.T) {
	got := plsqlFunctions(`
CREATE OR REPLACE FUNCTION check_status(p_id NUMBER) RETURN VARCHAR2 IS
BEGIN
    IF p_id > 0 THEN
        RETURN 'VALID';
    END IF;
    RETURN 'INVALID';
END check_status;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 2 {
		t.Errorf("CCN = %d, want 2", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_CCN_IfElsif(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE test_proc IS
BEGIN
    IF x = 1 THEN
        NULL;
    ELSIF x = 2 THEN
        NULL;
    ELSIF x = 3 THEN
        NULL;
    ELSE
        NULL;
    END IF;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// 1 + IF + ELSIF + ELSIF = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_CCN_Case(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE test_case IS
BEGIN
    CASE status
        WHEN 'NEW' THEN
            NULL;
        WHEN 'PENDING' THEN
            NULL;
        WHEN 'COMPLETE' THEN
            NULL;
    END CASE;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// 1 + 3 WHENs = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_CCN_Loops(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE test_loops IS
BEGIN
    LOOP
        EXIT WHEN x > 10;
    END LOOP;
    WHILE x < 100 LOOP
        NULL;
    END LOOP;
    FOR i IN 1..10 LOOP
        NULL;
    END LOOP;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// 1 + LOOP + WHILE + FOR = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_CCN_LogicalOperators(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE test_logic IS
BEGIN
    IF x = 1 AND y = 2 OR z = 3 THEN
        NULL;
    END IF;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// 1 + IF + AND + OR = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_CCN_ExceptionHandling(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE test_exceptions IS
BEGIN
    NULL;
EXCEPTION
    WHEN NO_DATA_FOUND THEN
        NULL;
    WHEN TOO_MANY_ROWS THEN
        NULL;
    WHEN OTHERS THEN
        NULL;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	// 1 + 3 WHENs = 4
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_PLSQL_PackageProcedureAndFunction(t *testing.T) {
	got := plsqlFunctions(`
CREATE OR REPLACE PACKAGE BODY my_package IS
    PROCEDURE pkg_proc IS
    BEGIN
        NULL;
    END pkg_proc;

    FUNCTION pkg_func RETURN NUMBER IS
    BEGIN
        RETURN 1;
    END pkg_func;
END my_package;
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	if got[0].Name != "pkg_proc" {
		t.Errorf("got[0].Name = %q, want pkg_proc", got[0].Name)
	}
	if got[1].Name != "pkg_func" {
		t.Errorf("got[1].Name = %q, want pkg_func", got[1].Name)
	}
}

func TestUnit_PLSQL_NestedProcedure(t *testing.T) {
	got := plsqlFunctions(`
CREATE PROCEDURE outer_proc IS
    PROCEDURE inner_proc IS
    BEGIN
        NULL;
    END inner_proc;
BEGIN
    inner_proc;
END outer_proc;
`)
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
}

func TestUnit_PLSQL_FunctionWithAS(t *testing.T) {
	got := plsqlFunctions(`
CREATE FUNCTION get_total RETURN NUMBER AS
    v_total NUMBER;
BEGIN
    RETURN v_total;
END;
`)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d", len(got))
	}
	if got[0].Name != "get_total" {
		t.Errorf("name = %q, want get_total", got[0].Name)
	}
}
