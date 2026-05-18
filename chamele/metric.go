package chamele

// Metric describes a per-function numeric metric (NLOC, CCN, etc.).
// Built-in metrics live as fields on FunctionInfo; extension metrics live on
// FunctionInfo.Ext keyed by Name. The Get function hides that distinction
// from callers (warning filters, output formatters).
type Metric struct {
	Name    string
	Aliases []string
	Get     func(*FunctionInfo) int
}

// matches reports whether name equals Name or any Alias.
func (m Metric) matches(name string) bool {
	if m.Name == name {
		return true
	}
	for _, a := range m.Aliases {
		if a == name {
			return true
		}
	}
	return false
}

var builtInMetrics = []Metric{
	{Name: "nloc", Get: func(f *FunctionInfo) int { return f.NLOC }},
	{Name: "cyclomatic_complexity", Aliases: []string{"ccn"}, Get: func(f *FunctionInfo) int { return f.CyclomaticComplexity }},
	{Name: "token_count", Aliases: []string{"token"}, Get: func(f *FunctionInfo) int { return f.TokenCount }},
	{Name: "parameter_count", Aliases: []string{"parameter", "param"}, Get: func(f *FunctionInfo) int { return f.ParameterCount() }},
	{Name: "length", Get: func(f *FunctionInfo) int { return f.Length() }},
	{Name: "max_nesting_depth", Aliases: []string{"nd"}, Get: func(f *FunctionInfo) int { return f.MaxNestingDepth }},
	{Name: "fan_in", Get: func(f *FunctionInfo) int { return f.FanIn }},
	{Name: "fan_out", Get: func(f *FunctionInfo) int { return f.FanOut }},
	{Name: "general_fan_out", Get: func(f *FunctionInfo) int { return f.GeneralFanOut }},
}

// MetricByName returns the built-in Metric whose Name or any Alias matches.
func MetricByName(name string) (Metric, bool) {
	for _, m := range builtInMetrics {
		if m.matches(name) {
			return m, true
		}
	}
	return Metric{}, false
}
