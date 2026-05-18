package chamele

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/iszlai/chamele-go/internal/stringx"
)

// FunctionInfo holds all per-function complexity metrics collected during analysis.
// It mirrors Python's FunctionInfo class from lizard.py.
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

// NewFunctionInfo creates a FunctionInfo with seed values matching lizard's defaults.
func NewFunctionInfo(name, filename string, startLine int) *FunctionInfo {
	return &FunctionInfo{
		Name:                 name,
		LongName:             name,
		Filename:             filename,
		StartLine:            startLine,
		EndLine:              startLine,
		CyclomaticComplexity: 1,
		NLOC:                 1,
		TokenCount:           1,
		TopNestingLevel:      -1,
	}
}

// Length returns EndLine - StartLine + 1.
func (f *FunctionInfo) Length() int { return f.EndLine - f.StartLine + 1 }

// Location returns a human-readable position string, matching Python's location property.
func (f *FunctionInfo) Location() string {
	return " " + f.Name + "@" +
		strconv.Itoa(f.StartLine) + "-" +
		strconv.Itoa(f.EndLine) + "@" +
		f.Filename
}

// UnqualifiedName returns the last "::" separated segment of the function name.
func (f *FunctionInfo) UnqualifiedName() string {
	parts := strings.Split(f.Name, "::")
	return parts[len(parts)-1]
}

// paramRe extracts the bare parameter name, stripping type annotations (`: …`)
// and default values (`= …`). Both optional groups require a leading space to
// avoid false positives on things like "char *p".
var paramRe = regexp.MustCompile(`(\w+)(\s=.*)?(\s:.*)?$`)

// Parameters returns the bare parameter names, excluding empty entries from
// trailing commas. Mirrors Python's FunctionInfo.parameters property.
func (f *FunctionInfo) Parameters() []string {
	var result []string
	for _, p := range f.FullParameters {
		m := paramRe.FindStringSubmatch(p)
		if m != nil {
			result = append(result, m[1])
		}
	}
	return result
}

// ParameterCount returns len(Parameters()).
func (f *FunctionInfo) ParameterCount() int { return len(f.Parameters()) }

// AddToFunctionName appends app to both Name and LongName.
func (f *FunctionInfo) AddToFunctionName(app string) {
	f.Name += app
	f.LongName += app
}

// AddToLongName appends app to LongName, inserting a space when both the
// current tail and app start with a letter (to keep words readable).
func (f *FunctionInfo) AddToLongName(app string) {
	if f.LongName != "" && len(app) > 0 {
		last := f.LongName[len(f.LongName)-1]
		if stringx.IsAlpha(last) && stringx.IsAlpha(app[0]) {
			f.LongName += " "
		}
	}
	f.LongName += app
}

// GetInt returns the int value at key in f.Ext, or 0 if missing or non-int.
func (f *FunctionInfo) GetInt(key string) int {
	if f.Ext == nil {
		return 0
	}
	if v, ok := f.Ext[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return 0
}

// Inc increments the int value at key in f.Ext by delta, creating the entry
// (and the Ext map) if needed.
func (f *FunctionInfo) Inc(key string, delta int) {
	if f.Ext == nil {
		f.Ext = make(map[string]any)
	}
	cur, _ := f.Ext[key].(int)
	f.Ext[key] = cur + delta
}

// AddParameter records one parameter token. A "," starts a new parameter slot;
// all other tokens are appended (with a space) to the current slot.
func (f *FunctionInfo) AddParameter(tok string) {
	f.AddToLongName(" " + tok)
	switch {
	case len(f.FullParameters) == 0:
		f.FullParameters = append(f.FullParameters, tok)
	case tok == ",":
		f.FullParameters = append(f.FullParameters, "")
	default:
		f.FullParameters[len(f.FullParameters)-1] += " " + tok
	}
}
