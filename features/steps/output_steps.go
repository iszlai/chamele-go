package steps

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog"
	"github.com/iszlai/chamele-go/chamele"
	"github.com/iszlai/chamele-go/output"
)

func registerOutputSteps(sc *godog.ScenarioContext, w *World) {
	// Function-level assertions
	sc.Step(`^the function "([^"]+)" should have CCN (\d+)$`, func(name string, ccn int) error {
		fn := w.FunctionByName(name)
		if fn == nil {
			return fmt.Errorf("function %q not found; available: %v", name, functionNames(w))
		}
		if fn.CyclomaticComplexity != ccn {
			return fmt.Errorf("function %q CCN = %d, want %d", name, fn.CyclomaticComplexity, ccn)
		}
		return nil
	})

	sc.Step(`^the function "([^"]+)" should have (\d+) parameters?$`, func(name string, count int) error {
		fn := w.FunctionByName(name)
		if fn == nil {
			return fmt.Errorf("function %q not found; available: %v", name, functionNames(w))
		}
		if fn.ParameterCount() != count {
			return fmt.Errorf("function %q parameter count = %d, want %d", name, fn.ParameterCount(), count)
		}
		return nil
	})

	sc.Step(`^the function "([^"]+)" should have NLOC (\d+)$`, func(name string, nloc int) error {
		fn := w.FunctionByName(name)
		if fn == nil {
			return fmt.Errorf("function %q not found; available: %v", name, functionNames(w))
		}
		if fn.NLOC != nloc {
			return fmt.Errorf("function %q NLOC = %d, want %d", name, fn.NLOC, nloc)
		}
		return nil
	})

	// Count assertions
	sc.Step(`^(\d+) functions? should be detected$`, func(count int) error {
		got := len(w.AllFunctions())
		if got != count {
			return fmt.Errorf("expected %d functions, got %d: %v", count, got, functionNames(w))
		}
		return nil
	})

	sc.Step(`^(\d+) files? should be reported$`, func(count int) error {
		// Only count files that actually parsed (have non-empty filename and were processed).
		got := 0
		for _, fi := range w.Results {
			if fi.Filename != "" {
				got++
			}
		}
		if got != count {
			return fmt.Errorf("expected %d files, got %d", count, got)
		}
		return nil
	})

	sc.Step(`^no functions? should be detected$`, func() error {
		got := len(w.AllFunctions())
		if got != 0 {
			return fmt.Errorf("expected 0 functions, got %d: %v", got, functionNames(w))
		}
		return nil
	})

	// Rendering steps
	sc.Step(`^I render the result as CSV$`, func() error {
		var b bytes.Buffer
		output.PrintCSV(&b, w.Results, false)
		w.Rendered = b.String()
		return nil
	})
	sc.Step(`^I render the result as XML$`, func() error {
		var b bytes.Buffer
		output.PrintXML(&b, w.Results, false)
		w.Rendered = b.String()
		return nil
	})
	sc.Step(`^I render the result as HTML$`, func() error {
		var b bytes.Buffer
		output.PrintHTML(&b, w.Results)
		w.Rendered = b.String()
		return nil
	})
	sc.Step(`^I render the result as tabular$`, func() error {
		var b bytes.Buffer
		output.PrintTabular(&b, w.Results, output.TabularOptions{
			Thresholds: []chamele.Threshold{
				{Metric: "cyclomatic_complexity", Limit: 15},
				{Metric: "length", Limit: 1000},
				{Metric: "parameter_count", Limit: 100},
			},
		})
		w.Rendered = b.String()
		return nil
	})
	sc.Step(`^I render the result as checkstyle$`, func() error {
		var b bytes.Buffer
		output.PrintCheckstyle(&b, w.Results, []chamele.Threshold{
			{Metric: "cyclomatic_complexity", Limit: 5},
		})
		w.Rendered = b.String()
		return nil
	})
	sc.Step(`^the output should contain "([^"]+)"$`, func(needle string) error {
		if !strings.Contains(w.Rendered, needle) {
			return fmt.Errorf("output does not contain %q; got:\n%s", needle, w.Rendered)
		}
		return nil
	})

	// CLI assertions
	sc.Step(`^the exit code should be (\d+)$`, func(code string) error {
		want, _ := strconv.Atoi(code)
		if w.CLIExitCode != want {
			return fmt.Errorf("exit code = %d, want %d", w.CLIExitCode, want)
		}
		return nil
	})
}

func functionNames(w *World) []string {
	fns := w.AllFunctions()
	names := make([]string, len(fns))
	for i, f := range fns {
		names[i] = f.Name
	}
	return names
}
