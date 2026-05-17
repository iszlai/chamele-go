package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintMSVSWarnings prints MSVS-format warnings for functions exceeding thresholds.
func PrintMSVSWarnings(w io.Writer, files []chamele.FileInformation, thresholds []chamele.Threshold) int {
	warnings := chamele.WarningFilter(files, thresholds)
	for _, fn := range warnings {
		fmt.Fprintf(w, "%s(%d): warning: %s (%s) has NLOC %d, CCN %d, token %d, PARAM %d, length %d\n",
			fn.Filename, fn.StartLine,
			fn.Name, fn.LongName,
			fn.NLOC, fn.CyclomaticComplexity, fn.TokenCount, fn.ParameterCount(), fn.Length(),
		)
	}
	return len(warnings)
}
