package output

import "github.com/iszlai/chamele-go/chamele"

// eachFunction calls fn for every function in every non-empty file. Output
// formatters use it to skip the standard "for i := range files { fi.IsEmpty()
// → continue; for _, fn := ... }" boilerplate.
func eachFunction(files []chamele.FileInformation, fn func(fi *chamele.FileInformation, f *chamele.FunctionInfo)) {
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		for _, f := range fi.Functions {
			fn(fi, f)
		}
	}
}

// eachFile calls fn for every non-empty file.
func eachFile(files []chamele.FileInformation, fn func(fi *chamele.FileInformation)) {
	for i := range files {
		fi := &files[i]
		if fi.IsEmpty() {
			continue
		}
		fn(fi)
	}
}
