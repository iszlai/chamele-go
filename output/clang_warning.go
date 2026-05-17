package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintClangWarnings prints clang-format warnings for functions exceeding thresholds.
func PrintClangWarnings(w io.Writer, files []chamele.FileInformation, thresholds []chamele.Threshold) int {
	warnings := chamele.WarningFilter(files, thresholds)
	for _, fn := range warnings {
		fmt.Fprintf(w, "%s warning: %s has %d NLOC, %d CCN, %d token, %d PARAM, %d length\n",
			fn.Location(),
			fn.Name,
			fn.NLOC,
			fn.CyclomaticComplexity,
			fn.TokenCount,
			fn.ParameterCount(),
			fn.Length(),
		)
	}
	return len(warnings)
}
