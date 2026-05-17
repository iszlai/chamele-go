package chamele

import (
	"os"
	"strings"
)

// Threshold describes a metric name and its maximum allowed value.
type Threshold struct {
	Metric string
	Limit  int
}

// WarningFilter yields functions from fileInfos that exceed any threshold,
// unless all exceeded metrics are in the function's ForgivenMetrics set.
func WarningFilter(fileInfos []FileInformation, thresholds []Threshold) []*FunctionInfo {
	var warnings []*FunctionInfo
	for i := range fileInfos {
		for _, fn := range fileInfos[i].Functions {
			violated := violatedThresholds(fn, thresholds)
			if len(violated) == 0 {
				continue
			}
			if allForgiven(fn, violated) {
				continue
			}
			warnings = append(warnings, fn)
		}
	}
	return warnings
}

func violatedThresholds(fn *FunctionInfo, thresholds []Threshold) []string {
	var out []string
	for _, t := range thresholds {
		val := metricValue(fn, t.Metric)
		if val > t.Limit {
			out = append(out, t.Metric)
		}
	}
	return out
}

func metricValue(fn *FunctionInfo, metric string) int {
	switch metric {
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
	case "max_nesting_depth", "nd":
		return fn.MaxNestingDepth
	}
	return 0
}

func allForgiven(fn *FunctionInfo, metrics []string) bool {
	for _, m := range metrics {
		if _, ok := fn.ForgivenMetrics[m]; !ok {
			return false
		}
	}
	return true
}

// whitelistItem is a parsed entry from whitelizard.txt.
type whitelistItem struct {
	filename      string // empty means match any file
	functionNames []string
}

// WhitelistFilter removes functions whose name appears in the whitelist file.
func WhitelistFilter(warnings []*FunctionInfo, whitelistPath string) []*FunctionInfo {
	script := readWhitelist(whitelistPath)
	items := parseWhitelist(script)
	var out []*FunctionInfo
	for _, w := range warnings {
		if !inWhitelist(w, items) {
			out = append(out, w)
		}
	}
	return out
}

func readWhitelist(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func parseWhitelist(script string) []whitelistItem {
	var items []whitelistItem
	for _, line := range strings.Split(script, "\n") {
		line = strings.SplitN(line, "#", 2)[0] // strip comments
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		item := whitelistItem{}
		// Replace :: with ## to avoid splitting on : used in filenames
		normalized := strings.ReplaceAll(line, "::", "##")
		parts := strings.SplitN(normalized, ":", 2)
		if len(parts) == 2 {
			item.filename = parts[0]
			line = parts[1]
		}
		for _, name := range strings.Split(line, ",") {
			name = strings.TrimSpace(strings.ReplaceAll(name, "##", "::"))
			if name != "" {
				item.functionNames = append(item.functionNames, name)
			}
		}
		items = append(items, item)
	}
	return items
}

func inWhitelist(fn *FunctionInfo, items []whitelistItem) bool {
	for _, item := range items {
		fileMatch := item.filename == "" || item.filename == fn.Filename
		for _, name := range item.functionNames {
			if fileMatch && name == fn.Name {
				return true
			}
		}
	}
	return false
}
