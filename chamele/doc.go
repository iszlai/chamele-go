// Package chamele provides a Go port of the lizard source-code complexity
// analyzer. It computes NLOC, cyclomatic complexity (CCN), token count,
// parameter count, and function length for 27+ programming languages without
// requiring a full AST or import resolution.
//
// Typical usage:
//
//	import _ "github.com/iszlai/chamele-go/languages/all"
//
//	results, err := chamele.Analyze([]string{"./myproject"})
package chamele
