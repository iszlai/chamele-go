package chamele

// FileInfoBuilder is the analysis context passed to processors and language
// readers. It accumulates tokens into FunctionInfo entries and assembles the
// final FileInformation.
//
// The embedded *NestingStack gives callers direct access to AddBareNesting,
// AddNamespace, StartNewFunctionNesting, PopNesting, WithNamespace, etc.,
// mirroring Python's FileInfoBuilder.__getattr__ delegation.
type FileInfoBuilder struct {
	*NestingStack
	fileinfo        FileInformation
	CurrentLine     int
	Forgive         bool
	ForgiveGlobal   bool
	Newline         bool
	globalFunction  *FunctionInfo
	CurrentFunction *FunctionInfo
	stackedFunctions []*FunctionInfo
}

// NewFileInfoBuilder creates a builder for the given filename.
func NewFileInfoBuilder(filename string) *FileInfoBuilder {
	global := NewFunctionInfo("*global*", filename, 0)
	return &FileInfoBuilder{
		NestingStack:    &NestingStack{},
		fileinfo:        FileInformation{Filename: filename},
		globalFunction:  global,
		CurrentFunction: global,
		Newline:         true,
	}
}

// CurrentFunctionLongName returns the long name of the current function.
// This is needed by language readers that form function-pointer names from
// the long name of a preceding declaration.
func (b *FileInfoBuilder) CurrentFunctionLongName() string {
	return b.CurrentFunction.LongName
}

// Build finalises and returns the completed FileInformation.
func (b *FileInfoBuilder) Build() *FileInformation {
	fi := b.fileinfo
	return &fi
}

// AddNLOC increments the file and function NLOC counters.
func (b *FileInfoBuilder) AddNLOC(count int) {
	b.fileinfo.NLOC += count
	b.CurrentFunction.NLOC += count
	b.CurrentFunction.EndLine = b.CurrentLine
	b.Newline = count > 0
}

// TryNewFunction creates a candidate function at the current line with the
// given name, qualified by any enclosing namespaces.
func (b *FileInfoBuilder) TryNewFunction(name string) {
	b.CurrentFunction = NewFunctionInfo(
		b.WithNamespace(name),
		b.fileinfo.Filename,
		b.CurrentLine,
	)
	b.CurrentFunction.TopNestingLevel = b.CurrentNestingLevel()
}

// ConfirmNewFunction locks in the current candidate as a real function.
func (b *FileInfoBuilder) ConfirmNewFunction() {
	b.StartNewFunctionNesting(b.CurrentFunction)
	b.CurrentFunction.CyclomaticComplexity = 1
}

// RestartNewFunction combines TryNewFunction and ConfirmNewFunction.
func (b *FileInfoBuilder) RestartNewFunction(name string) {
	b.TryNewFunction(name)
	b.ConfirmNewFunction()
}

// PushNewFunction stacks the current function and starts a new one.
func (b *FileInfoBuilder) PushNewFunction(name string) {
	b.stackedFunctions = append(b.stackedFunctions, b.CurrentFunction)
	b.RestartNewFunction(name)
}

// AddCondition increments the cyclomatic complexity counter.
func (b *FileInfoBuilder) AddCondition(inc int) {
	b.CurrentFunction.CyclomaticComplexity += inc
}

// AddToLongFunctionName appends app to the current function's long name.
func (b *FileInfoBuilder) AddToLongFunctionName(app string) {
	b.CurrentFunction.AddToLongName(app)
}

// AddToFunctionName appends app to the current function's name.
func (b *FileInfoBuilder) AddToFunctionName(app string) {
	b.CurrentFunction.AddToFunctionName(app)
}

// Parameter records one parameter token for the current function.
func (b *FileInfoBuilder) Parameter(tok string) {
	b.CurrentFunction.AddParameter(tok)
}

// PopNesting pops one nesting level. If it was a function, EndOfFunction is called.
func (b *FileInfoBuilder) PopNesting() {
	item := b.NestingStack.PopNesting()
	if item == nil {
		return
	}
	if item.kind == nkFunction {
		endLine := b.CurrentFunction.EndLine
		b.endOfFunction()
		if last := b.NestingStack.LastFunction(); last != nil {
			b.CurrentFunction = last
		} else {
			b.CurrentFunction = b.globalFunction
		}
		b.CurrentFunction.EndLine = endLine
	}
}

// endOfFunction finalises the current function and appends it to the file list
// (unless forgiven). Mirrors Python's FileInfoBuilder.end_of_function.
func (b *FileInfoBuilder) endOfFunction() {
	if !b.Forgive {
		if b.CurrentFunction.Name != "*global*" || !b.ForgiveGlobal {
			b.fileinfo.Functions = append(b.fileinfo.Functions, b.CurrentFunction)
		}
	}
	b.Forgive = false
	if len(b.stackedFunctions) > 0 {
		b.CurrentFunction = b.stackedFunctions[len(b.stackedFunctions)-1]
		b.stackedFunctions = b.stackedFunctions[:len(b.stackedFunctions)-1]
	} else {
		b.CurrentFunction = b.globalFunction
	}
}
