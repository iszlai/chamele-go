package java_test

import (
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/languages/java"
)

func javaFunctions(src string) []*chamele.FunctionInfo {
	r := java.NewJavaReader()
	a := chamele.NewFileAnalyzer()
	return a.AnalyzeSourceCode("test.java", []byte(src), r).Functions
}

func names(fns []*chamele.FunctionInfo) []string {
	out := make([]string, len(fns))
	for i, f := range fns {
		out[i] = f.Name
	}
	return out
}

func TestUnit_Java_SimpleFunction(t *testing.T) {
	got := javaFunctions("void fun() {}")
	if len(got) != 1 || got[0].Name != "fun" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Java_FunctionWithThrows(t *testing.T) {
	got := javaFunctions("void fun() throws e1, e2{}")
	if len(got) != 1 {
		t.Errorf("expected 1, got %d: %v", len(got), names(got))
	}
}

func TestUnit_Java_FunctionWithDecorator(t *testing.T) {
	got := javaFunctions("@abc() void fun() throws e1, e2{}")
	if len(got) != 1 || got[0].Name != "fun" {
		t.Errorf("got %v", names(got))
	}
}

func TestUnit_Java_AbstractNotCounted(t *testing.T) {
	got := javaFunctions("abstract void fun(); void fun1(){}")
	if len(got) != 1 || got[0].Name != "fun1" {
		t.Errorf("expected [fun1], got %v", names(got))
	}
}

func TestUnit_Java_AbstractWithThrowsNotCounted(t *testing.T) {
	got := javaFunctions("abstract void fun() throws e; void fun2(){}")
	if len(got) != 1 || got[0].Name != "fun2" {
		t.Errorf("expected [fun2], got %v", names(got))
	}
}

func TestUnit_Java_ClassMethod(t *testing.T) {
	got := javaFunctions("class A extends B { void f(){}}")
	if len(got) != 1 || got[0].Name != "A::f" {
		t.Errorf("expected A::f, got %v", names(got))
	}
}

func TestUnit_Java_ClassWithInterface(t *testing.T) {
	got := javaFunctions("class A implements B { void f(){}}")
	if len(got) != 1 || got[0].Name != "A::f" {
		t.Errorf("expected A::f, got %v", names(got))
	}
}

func TestUnit_Java_ClassWithDecorator(t *testing.T) {
	got := javaFunctions("@abc() class funxx{ }")
	if len(got) != 0 {
		t.Errorf("expected 0 functions, got %v", names(got))
	}
}

func TestUnit_Java_CCN_IfForWhile(t *testing.T) {
	got := javaFunctions("void f() { if(a) {} for(;;) {} while(b) {} }")
	if len(got) != 1 {
		t.Fatalf("expected 1 function, got %d", len(got))
	}
	if got[0].CyclomaticComplexity != 4 {
		t.Errorf("CCN = %d, want 4", got[0].CyclomaticComplexity)
	}
}

func TestUnit_Java_TransactionalAnnotationIssue(t *testing.T) {
	code := `
public class LizardTest {
    @Transactional(rollbackFor = Exception.class)
    public void test1() {
        if (a) {}
        if (b) {}
    }
}`
	got := javaFunctions(code)
	if len(got) != 1 {
		t.Fatalf("expected 1, got %d: %v", len(got), names(got))
	}
	if got[0].Name != "LizardTest::test1" {
		t.Errorf("name = %q, want LizardTest::test1", got[0].Name)
	}
}
