// Package all registers every built-in extension.
// Import it for side effects:
//
//	import _ "github.com/iszlai/chamele-go/ext/all"
package all

import (
	_ "github.com/iszlai/chamele-go/ext/boolcount"
	_ "github.com/iszlai/chamele-go/ext/complextags"
	_ "github.com/iszlai/chamele-go/ext/cpre"
	_ "github.com/iszlai/chamele-go/ext/dependencycount"
	_ "github.com/iszlai/chamele-go/ext/dumpcomments"
	_ "github.com/iszlai/chamele-go/ext/duplicate"
	_ "github.com/iszlai/chamele-go/ext/dupparams"
	_ "github.com/iszlai/chamele-go/ext/exitcount"
	_ "github.com/iszlai/chamele-go/ext/gotocount"
	_ "github.com/iszlai/chamele-go/ext/ignoreassert"
	_ "github.com/iszlai/chamele-go/ext/io"
	_ "github.com/iszlai/chamele-go/ext/mccabe"
	_ "github.com/iszlai/chamele-go/ext/modified"
	_ "github.com/iszlai/chamele-go/ext/nd"
	_ "github.com/iszlai/chamele-go/ext/nonstrict"
	_ "github.com/iszlai/chamele-go/ext/ns"
	_ "github.com/iszlai/chamele-go/ext/outside"
	_ "github.com/iszlai/chamele-go/ext/statementcount"
	_ "github.com/iszlai/chamele-go/ext/wordcount"
)
