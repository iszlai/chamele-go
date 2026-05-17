package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintHTML writes a minimal HTML report. Data-equivalent to lizard's html_output;
// CSS/layout may differ (see docs/divergences.md).
func PrintHTML(w io.Writer, files []chamele.FileInformation) {
	fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="utf-8">`)
	fmt.Fprintln(w, `<title>chamele analysis</title></head><body>`)
	fmt.Fprintln(w, `<table border="1"><tr><th>NLOC</th><th>CCN</th><th>token</th><th>PARAM</th><th>length</th><th>location</th></tr>`)
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		for _, fn := range fi.Functions {
			fmt.Fprintf(w,
				"<tr><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%s</td></tr>\n",
				fn.NLOC, fn.CyclomaticComplexity, fn.TokenCount, fn.ParameterCount(), fn.Length(),
				xmlEsc(fn.Location()),
			)
		}
	}
	fmt.Fprintln(w, `</table></body></html>`)
}
