package steps

import "github.com/iszlai/chamele-go/chamele"

// World holds shared state for a single BDD scenario.
type World struct {
	SourceCode  string
	Results     []chamele.FileInformation
	CLIStdout   string
	CLIStderr   string
	CLIExitCode int
}

// Reset clears all state between scenarios.
func (w *World) Reset() {
	*w = World{}
}
