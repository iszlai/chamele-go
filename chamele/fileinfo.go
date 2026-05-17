package chamele

// FileInformation holds per-file complexity metrics produced by Analyze.
type FileInformation struct {
	Filename   string
	NLOC       int
	TokenCount int
	Functions  []*FunctionInfo
	WordCount  map[string]int
}

// IsEmpty reports whether this represents a failed or skipped file.
func (fi *FileInformation) IsEmpty() bool { return fi.Filename == "" }

// FunctionCount returns the number of functions found.
func (fi *FileInformation) FunctionCount() int { return len(fi.Functions) }

// AverageNLOC returns the mean NLOC across all functions.
func (fi *FileInformation) AverageNLOC() float64 { return fi.functionsAverage("nloc") }

// AverageTokenCount returns the mean token count across all functions.
func (fi *FileInformation) AverageTokenCount() float64 { return fi.functionsAverage("token_count") }

// AverageCCN returns the mean cyclomatic complexity across all functions.
func (fi *FileInformation) AverageCCN() float64 { return fi.functionsAverage("ccn") }

// CCN returns the sum of cyclomatic complexities across all functions.
func (fi *FileInformation) CCN() int {
	sum := 0
	for _, f := range fi.Functions {
		sum += f.CyclomaticComplexity
	}
	return sum
}

func (fi *FileInformation) functionsAverage(field string) float64 {
	if len(fi.Functions) == 0 {
		return 0
	}
	var sum float64
	for _, f := range fi.Functions {
		switch field {
		case "nloc":
			sum += float64(f.NLOC)
		case "token_count":
			sum += float64(f.TokenCount)
		case "ccn":
			sum += float64(f.CyclomaticComplexity)
		}
	}
	return sum / float64(len(fi.Functions))
}
