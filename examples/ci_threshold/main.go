// Example: fail CI if any function exceeds CCN threshold.
package main

import (
	"fmt"
	"os"

	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/languages/all"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, err := chamele.Analyze([]string{dir})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	warnings := chamele.WarningFilter(files, []chamele.Threshold{
		{Metric: "cyclomatic_complexity", Limit: 15},
	})
	for _, fn := range warnings {
		fmt.Printf("WARNING: %s has CCN %d > 15\n", fn.Location(), fn.CyclomaticComplexity)
	}
	if len(warnings) > 0 {
		os.Exit(1)
	}
}
