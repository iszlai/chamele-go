package steps

import (
	"github.com/iszlai/chamele-go/chamele"
)

// World holds shared state for a single BDD scenario.
type World struct {
	SourceCode  string
	Lang        string
	Results     []chamele.FileInformation
	CLIStdout   string
	CLIStderr   string
	CLIExitCode int
}

// Reset clears all state between scenarios.
func (w *World) Reset() {
	*w = World{Lang: "go"}
}

// AllFunctions returns a flat list of all detected functions.
func (w *World) AllFunctions() []*chamele.FunctionInfo {
	var all []*chamele.FunctionInfo
	for i := range w.Results {
		all = append(all, w.Results[i].Functions...)
	}
	return all
}

// FunctionByName finds a function by name across all results.
func (w *World) FunctionByName(name string) *chamele.FunctionInfo {
	for _, fn := range w.AllFunctions() {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}
