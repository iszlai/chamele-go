package chamele

// FileInfoBuilder constructs a FileInformation incrementally as tokens are
// processed by the analysis pipeline.
type FileInfoBuilder struct {
	filename        string
	currentFunction *FunctionInfo
	functions       []*FunctionInfo
	nesting         NestingStack
}

// NewFileInfoBuilder creates a builder for the given file.
func NewFileInfoBuilder(filename string) *FileInfoBuilder {
	return &FileInfoBuilder{filename: filename}
}

// Build finalises and returns the FileInformation.
func (b *FileInfoBuilder) Build() *FileInformation {
	fi := &FileInformation{
		Filename:  b.filename,
		Functions: b.functions,
	}
	return fi
}
