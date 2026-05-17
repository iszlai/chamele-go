package chamele

import (
	"fmt"
	"strings"
)

// ColumnItem describes one output column in the tabular report.
type ColumnItem struct {
	Caption    string
	Value      string // field name or key for FunctionInfo
	AvgCaption string // non-empty → average row shown
}

// OutputScheme collects the column schema for a given set of extensions.
// It matches the structure of Python's OutputScheme class.
type OutputScheme struct {
	Items []ColumnItem
}

// NewOutputScheme builds the standard column list, extended by any registered extensions.
func NewOutputScheme(exts []Extension) *OutputScheme {
	items := []ColumnItem{
		{Caption: "  NLOC  ", Value: "nloc", AvgCaption: " Avg.NLOC "},
		{Caption: "  CCN  ", Value: "cyclomatic_complexity", AvgCaption: " AvgCCN "},
		{Caption: " token ", Value: "token_count", AvgCaption: " Avg.token "},
		{Caption: " PARAM ", Value: "parameter_count"},
		{Caption: " length ", Value: "length"},
	}
	for _, ext := range exts {
		for _, col := range ext.FunctionInfoColumns() {
			items = append(items, ColumnItem{Caption: col.Header, Value: col.Header})
		}
	}
	items = append(items, ColumnItem{Caption: " location  ", Value: "location"})
	return &OutputScheme{Items: items}
}

// Captions returns the concatenated column header string.
func (s *OutputScheme) Captions() string {
	var b strings.Builder
	for _, item := range s.Items {
		b.WriteString(item.Caption)
	}
	return b.String()
}

func head(captions string) string {
	line := strings.Repeat("=", len(captions))
	dash := strings.Repeat("-", len(captions))
	return line + "\n" + captions + "\n" + dash
}

// AverageCaptions returns the concatenated average column header string.
func (s *OutputScheme) AverageCaptions() string {
	var b strings.Builder
	for _, item := range s.Items {
		if item.AvgCaption != "" {
			b.WriteString(item.AvgCaption)
		}
	}
	return b.String()
}

// FunctionInfoHead returns the header block (=====, captions, -----).
func (s *OutputScheme) FunctionInfoHead() string { return head(s.Captions()) }

// FunctionInfoLine formats one function row, right-justified to column widths.
func (s *OutputScheme) FunctionInfoLine(fn *FunctionInfo) string {
	var b strings.Builder
	for _, item := range s.Items {
		if item.Caption == "" {
			continue
		}
		val := functionFieldStr(fn, item.Value)
		b.WriteString(fmt.Sprintf("%*s", len(item.Caption), val))
	}
	return b.String()
}

func functionFieldStr(fn *FunctionInfo, field string) string {
	switch field {
	case "nloc":
		return fmt.Sprintf("%d", fn.NLOC)
	case "cyclomatic_complexity":
		return fmt.Sprintf("%d", fn.CyclomaticComplexity)
	case "token_count":
		return fmt.Sprintf("%d", fn.TokenCount)
	case "parameter_count":
		return fmt.Sprintf("%d", fn.ParameterCount())
	case "length":
		return fmt.Sprintf("%d", fn.Length())
	case "location":
		return fn.Location()
	}
	return ""
}
