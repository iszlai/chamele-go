// Package output implements output formatters for chamele analysis results.
package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/iszlai/chamele-go/chamele"
)

// PrintTabular writes the default tabular report (matches lizard's print_result).
// Returns the number of warning functions.
func PrintTabular(w io.Writer, files []chamele.FileInformation, opts TabularOptions) int {
	scheme := chamele.NewOutputScheme(nil)
	saved := printModules(w, files, scheme, opts.Verbose)
	warnings := chamele.WarningFilter(saved, opts.Thresholds)
	if opts.Whitelist != "" {
		warnings = chamele.WhitelistFilter(warnings, opts.Whitelist)
	}
	if len(opts.Sort) > 0 {
		warnings = sortWarnings(warnings, opts.Sort)
	}
	warnCount, warnNLOC := printWarnings(w, warnings, scheme, opts.Thresholds)
	printTotal(w, warnCount, warnNLOC, saved, scheme)
	return warnCount
}

// TabularOptions configures the tabular printer.
type TabularOptions struct {
	Thresholds []chamele.Threshold
	Sort       []string
	Whitelist  string
	Verbose    bool
}

func printModules(w io.Writer, files []chamele.FileInformation, scheme *chamele.OutputScheme, verbose bool) []chamele.FileInformation {
	_, _ = fmt.Fprintln(w, scheme.FunctionInfoHead())
	var saved []chamele.FileInformation
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		saved = append(saved, *fi)
		for _, fn := range fi.Functions {
			_, _ = fmt.Fprintln(w, scheme.FunctionInfoLine(fn))
		}
	}
	_, _ = fmt.Fprintf(w, "%d file analyzed.\n", len(saved))
	_, _ = fmt.Fprintln(w, strings.Repeat("=", 62))
	_, _ = fmt.Fprintln(w, "NLOC   "+scheme.AverageCaptions()+" function_cnt    file")
	_, _ = fmt.Fprintln(w, strings.Repeat("-", 62))
	for i := range saved {
		fi := &saved[i]
		_, _ = fmt.Fprintf(w, "%7d%s%10d     %s\n",
			fi.NLOC,
			formatAverages(fi, scheme),
			len(fi.Functions),
			fi.Filename,
		)
	}
	return saved
}

func formatAverages(fi *chamele.FileInformation, scheme *chamele.OutputScheme) string {
	return fmt.Sprintf(" %8.1f %7.1f %9.1f",
		fi.AverageNLOC(), fi.AverageCCN(), fi.AverageTokenCount())
}

func printWarnings(w io.Writer, warnings []*chamele.FunctionInfo, scheme *chamele.OutputScheme, thresholds []chamele.Threshold) (int, int) {
	if len(warnings) == 0 {
		parts := make([]string, len(thresholds))
		for i, t := range thresholds {
			parts[i] = fmt.Sprintf("%s > %d", t.Metric, t.Limit)
		}
		msg := "No thresholds exceeded (" + strings.Join(parts, " or ") + ")"
		_, _ = fmt.Fprintln(w, "\n"+strings.Repeat("=", len(msg))+"\n"+msg)
		return 0, 0
	}
	parts := make([]string, len(thresholds))
	for i, t := range thresholds {
		parts[i] = fmt.Sprintf("%s > %d", t.Metric, t.Limit)
	}
	warnStr := "!!!! Warnings (" + strings.Join(parts, " or ") + ") !!!!"
	_, _ = fmt.Fprintln(w, "\n"+strings.Repeat("=", len(warnStr))+"\n"+warnStr)
	_, _ = fmt.Fprintln(w, scheme.FunctionInfoHead())
	var warnCount, warnNLOC int
	for _, fn := range warnings {
		_, _ = fmt.Fprintln(w, scheme.FunctionInfoLine(fn))
		warnCount++
		warnNLOC += fn.NLOC
	}
	return warnCount, warnNLOC
}

func printTotal(w io.Writer, warnCount, warnNLOC int, files []chamele.FileInformation, scheme *chamele.OutputScheme) {
	allFns := collectFunctions(files)
	totalNLOC := 0
	for i := range files {
		totalNLOC += files[i].NLOC
	}
	nloc := totalNLOC
	nlocInFns := 0
	for _, fn := range allFns {
		nlocInFns += fn.NLOC
	}
	if nlocInFns == 0 {
		nlocInFns = 1
	}
	var avgCCN, avgNLOC, avgToken float64
	for _, fn := range allFns {
		avgCCN += float64(fn.CyclomaticComplexity)
		avgNLOC += float64(fn.NLOC)
		avgToken += float64(fn.TokenCount)
	}
	total := float64(max(len(allFns), 1))
	avgCCN /= total
	avgNLOC /= total
	avgToken /= total

	_, _ = fmt.Fprintln(w, strings.Repeat("=", 90))
	_, _ = fmt.Fprintln(w, "Total nloc  "+scheme.AverageCaptions()+"  Fun Cnt  Warning cnt   Fun Rt   nloc Rt")
	_, _ = fmt.Fprintln(w, strings.Repeat("-", 90))
	_, _ = fmt.Fprintf(w, "%10d %8.1f %7.1f %9.1f%9d%13d%10.2f%8.2f\n",
		nloc, avgNLOC, avgCCN, avgToken,
		len(allFns), warnCount,
		float64(warnCount)/float64(max(len(allFns), 1)),
		float64(warnNLOC)/float64(nlocInFns),
	)
}

func collectFunctions(files []chamele.FileInformation) []*chamele.FunctionInfo {
	var all []*chamele.FunctionInfo
	for i := range files {
		all = append(all, files[i].Functions...)
	}
	return all
}

func sortWarnings(warnings []*chamele.FunctionInfo, fields []string) []*chamele.FunctionInfo {
	if len(fields) == 0 {
		return warnings
	}
	// Simple insertion sort by first field descending.
	field := fields[0]
	for i := 1; i < len(warnings); i++ {
		for j := i; j > 0 && metricVal(warnings[j], field) > metricVal(warnings[j-1], field); j-- {
			warnings[j], warnings[j-1] = warnings[j-1], warnings[j]
		}
	}
	return warnings
}

func metricVal(fn *chamele.FunctionInfo, field string) int {
	switch field {
	case "cyclomatic_complexity", "ccn":
		return fn.CyclomaticComplexity
	case "nloc":
		return fn.NLOC
	case "token_count":
		return fn.TokenCount
	case "parameter_count":
		return fn.ParameterCount()
	case "length":
		return fn.Length()
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
