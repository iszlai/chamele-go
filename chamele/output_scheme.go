package chamele

import (
	"fmt"
	"strings"
)

// OutputScheme collects the column schema for a given set of extensions.
// It matches the structure of Python's OutputScheme class.
type OutputScheme struct {
	Items []ColumnItem
}

// metricColumn builds a ColumnItem from a built-in Metric and display strings.
func metricColumn(metricName, header, avgCaption string) ColumnItem {
	m, _ := MetricByName(metricName)
	return ColumnItem{
		Header:     header,
		AvgCaption: avgCaption,
		Value:      func(f *FunctionInfo) any { return m.Get(f) },
	}
}

// locationColumn is the trailing "location" column; not a numeric metric.
var locationColumn = ColumnItem{
	Header: " location  ",
	Value:  func(f *FunctionInfo) any { return f.Location() },
}

// NewOutputScheme builds the standard column list, extended by any registered extensions.
func NewOutputScheme(exts []Extension) *OutputScheme {
	items := []ColumnItem{
		metricColumn("nloc", "  NLOC  ", " Avg.NLOC "),
		metricColumn("cyclomatic_complexity", "  CCN  ", " AvgCCN "),
		metricColumn("token_count", " token ", " Avg.token "),
		metricColumn("parameter_count", " PARAM ", ""),
		metricColumn("length", " length ", ""),
	}
	for _, ext := range exts {
		items = append(items, ext.FunctionInfoColumns()...)
	}
	items = append(items, locationColumn)
	return &OutputScheme{Items: items}
}

// Captions returns the concatenated column header string.
func (s *OutputScheme) Captions() string {
	var b strings.Builder
	for _, item := range s.Items {
		b.WriteString(item.Header)
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
		if item.Header == "" || item.Value == nil {
			continue
		}
		val := fmt.Sprintf("%v", item.Value(fn))
		fmt.Fprintf(&b, "%*s", len(item.Header), val)
	}
	return b.String()
}
