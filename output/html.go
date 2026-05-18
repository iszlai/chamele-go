package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintHTML writes a minimal HTML report. Data-equivalent to lizard's html_output;
// CSS/layout may differ (see docs/divergences.md).
func PrintHTML(w io.Writer, files []chamele.FileInformation) {
	_, _ = fmt.Fprintln(w, `<!DOCTYPE html><html><head><meta charset="utf-8">`)
	_, _ = fmt.Fprintln(w, `<title>chamele analysis</title></head><body>`)
	_, _ = fmt.Fprintln(w, `<table border="1"><tr><th>NLOC</th><th>CCN</th><th>token</th><th>PARAM</th><th>length</th><th>location</th></tr>`)
	eachFunction(files, func(_ *chamele.FileInformation, fn *chamele.FunctionInfo) {
		_, _ = fmt.Fprintf(w,
			"<tr><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%s</td></tr>\n",
			fn.NLOC, fn.CyclomaticComplexity, fn.TokenCount, fn.ParameterCount(), fn.Length(),
			xmlEsc(fn.Location()),
		)
	})
	_, _ = fmt.Fprintln(w, `</table></body></html>`)
}
