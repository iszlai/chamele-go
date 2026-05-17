package output

import (
	"fmt"
	"io"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintCheckstyle writes Checkstyle-compatible XML for functions exceeding thresholds.
func PrintCheckstyle(w io.Writer, files []chamele.FileInformation, thresholds []chamele.Threshold) {
	fmt.Fprintln(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprintln(w, `<checkstyle version="4.3">`)
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
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
			fmt.Fprintf(w, "<file name=\"%s\">\n", xmlEsc(fi.Filename))
			for _, m := range msgs {
				fmt.Fprintln(w, m)
			}
			fmt.Fprintln(w, "</file>")
		}
	}
	fmt.Fprintln(w, "</checkstyle>")
}
