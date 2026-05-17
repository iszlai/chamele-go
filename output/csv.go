package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintCSV writes all functions in CSV format, matching lizard's csv_output.
func PrintCSV(w io.Writer, files []chamele.FileInformation, verbose bool) {
	if verbose {
		fmt.Fprintln(w, "NLOC,CCN,token,PARAM,length,location,file,function,long_name,start,end")
	}
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		for _, fn := range fi.Functions {
			loc := fmt.Sprintf("%s@%d-%d@%s",
				csvEscape(fn.Name), fn.StartLine, fn.EndLine, fi.Filename)
			fmt.Fprintf(w, "%d,%d,%d,%d,%d,%q,%q,%q,%q,%d,%d\n",
				fn.NLOC,
				fn.CyclomaticComplexity,
				fn.TokenCount,
				fn.ParameterCount(),
				fn.Length(),
				loc,
				fi.Filename,
				csvEscape(fn.Name),
				csvEscape(fn.LongName),
				fn.StartLine,
				fn.EndLine,
			)
		}
	}
}

func csvEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `'`)
}
