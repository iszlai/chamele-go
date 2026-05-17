package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintXML writes cppncss-compatible XML output, matching lizard's xml_output.
func PrintXML(w io.Writer, files []chamele.FileInformation, verbose bool) {
	_, _ = fmt.Fprintln(w, `<?xml version="1.0" ?>`)
	_, _ = fmt.Fprintln(w, `<?xml-stylesheet type="text/xsl" href="chamele.xsl"?>`)
	_, _ = fmt.Fprintln(w, `<cppncss>`)
	_, _ = fmt.Fprintln(w, `  <measure type="Function">`)
	_, _ = fmt.Fprintln(w, `    <labels><label>Nr.</label><label>NCSS</label><label>CCN</label></labels>`)

	num, totalNCSS, totalCCN := 0, 0, 0
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		for _, fn := range fi.Functions {
			num++
			totalNCSS += fn.NLOC
			totalCCN += fn.CyclomaticComplexity
			name := fn.Name
			if verbose {
				name = fn.LongName
			}
			label := xmlEsc(fmt.Sprintf("%s at line %d-%d@%s", name, fn.StartLine, fn.EndLine, fi.Filename))
			_, _ = fmt.Fprintf(w, "    <item name=\"%s\">\n", label)
			_, _ = fmt.Fprintf(w, "      <value>%d</value><value>%d</value><value>%d</value>\n",
				num, fn.NLOC, fn.CyclomaticComplexity)
			_, _ = fmt.Fprintln(w, "    </item>")
		}
	}
	if num > 0 {
		_, _ = fmt.Fprintf(w, "    <average label=\"NCSS\" value=\"%.2f\"/>\n", float64(totalNCSS)/float64(num))
		_, _ = fmt.Fprintf(w, "    <average label=\"CCN\" value=\"%.2f\"/>\n", float64(totalCCN)/float64(num))
	}
	_, _ = fmt.Fprintln(w, "  </measure>")

	_, _ = fmt.Fprintln(w, `  <measure type="File">`)
	_, _ = fmt.Fprintln(w, `    <labels><label>Nr.</label><label>NCSS</label><label>CCN</label><label>Functions</label></labels>`)
	allFns, allNLOC, allCCN, fcount := 0, 0, 0, 0
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		fcount++
		fileCCN := 0
		for _, fn := range fi.Functions {
			fileCCN += fn.CyclomaticComplexity
		}
		allFns += len(fi.Functions)
		allNLOC += fi.NLOC
		allCCN += fileCCN
		_, _ = fmt.Fprintf(w, "    <item name=\"%s\">\n", xmlEsc(fi.Filename))
		_, _ = fmt.Fprintf(w, "      <value>%d</value><value>%d</value><value>%d</value><value>%d</value>\n",
			fcount, fi.NLOC, fileCCN, len(fi.Functions))
		_, _ = fmt.Fprintln(w, "    </item>")
	}
	denom := max(fcount, 1)
	_, _ = fmt.Fprintf(w, "    <average label=\"NCSS\" value=\"%.2f\"/>\n", float64(allNLOC)/float64(denom))
	_, _ = fmt.Fprintf(w, "    <average label=\"CCN\" value=\"%.2f\"/>\n", float64(allCCN)/float64(denom))
	_, _ = fmt.Fprintf(w, "    <average label=\"Functions\" value=\"%.2f\"/>\n", float64(allFns)/float64(denom))
	_, _ = fmt.Fprintln(w, "  </measure>")
	_, _ = fmt.Fprintln(w, "</cppncss>")
}

func xmlEsc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}
