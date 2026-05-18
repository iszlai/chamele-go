package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintCheckstyle writes Checkstyle-compatible XML for functions exceeding thresholds.
func PrintCheckstyle(w io.Writer, files []chamele.FileInformation, thresholds []chamele.Threshold) {
	_, _ = fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	_, _ = fmt.Fprintln(w, `<checkstyle version="4.3">`)
	eachFile(files, func(fi *chamele.FileInformation) {
		var msgs []string
		for _, fn := range fi.Functions {
			for _, t := range thresholds {
				val := metricVal(fn, t.Metric)
				if val > t.Limit {
					msgs = append(msgs, fmt.Sprintf(
						`  <error line="%d" severity="warning" message="%s has %s %d"/>`,
						fn.StartLine, xmlEsc(fn.Name), t.Metric, val))
				}
			}
		}
		if len(msgs) > 0 {
			_, _ = fmt.Fprintf(w, "<file name=\"%s\">\n", xmlEsc(fi.Filename))
			for _, m := range msgs {
				_, _ = fmt.Fprintln(w, m)
			}
			_, _ = fmt.Fprintln(w, "</file>")
		}
	})
	_, _ = fmt.Fprintln(w, "</checkstyle>")
}
