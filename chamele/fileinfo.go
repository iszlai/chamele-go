package chamele

// FileInformation holds per-file complexity metrics.
type FileInformation struct {
	Filename          string
	NLOC              int
	AverageCCN        float64
	AverageNLOC       float64
	AverageTokenCount float64
	Functions         []*FunctionInfo
	WordCount         map[string]int
}

// IsEmpty reports whether this FileInformation represents a failed or skipped file.
func (fi *FileInformation) IsEmpty() bool {
	return fi.Filename == ""
}

// FunctionCount returns the number of functions found.
func (fi *FileInformation) FunctionCount() int { return len(fi.Functions) }
