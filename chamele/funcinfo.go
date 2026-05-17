package chamele

// FunctionInfo holds per-function complexity metrics.
type FunctionInfo struct {
	Name                 string
	LongName             string
	Filename             string
	StartLine            int
	EndLine              int
	CyclomaticComplexity int
	NLOC                 int
	TokenCount           int
	FullParameters       []string
	TopNestingLevel      int
	FanIn                int
	FanOut               int
	GeneralFanOut        int
	MaxNestingDepth      int
	ForgivenMetrics      map[string]struct{}
	Ext                  map[string]any
}

// Length returns the number of lines in the function (inclusive).
func (f *FunctionInfo) Length() int { return f.EndLine - f.StartLine + 1 }

// ParameterCount returns the number of parameters, excluding empty entries
// from trailing commas and stripping type annotations and default values.
func (f *FunctionInfo) ParameterCount() int {
	// Implemented in Phase 1.
	return len(f.FullParameters)
}

// UnqualifiedName returns the last segment of a "::" separated name.
func (f *FunctionInfo) UnqualifiedName() string {
	name := f.Name
	for i := len(name) - 1; i >= 0; i-- {
		if i > 0 && name[i-1] == ':' && name[i] == ':' {
			return name[i+1:]
		}
	}
	return name
}

// Location returns a human-readable location string.
func (f *FunctionInfo) Location() string {
	return " " + f.Name + "@" + itoa(f.StartLine) + "-" + itoa(f.EndLine) + "@" + f.Filename
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	if n < 0 {
		buf = append(buf, '-')
		n = -n
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		buf = append(buf, digits[i])
	}
	return string(buf)
}
