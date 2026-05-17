// Example: analyse a single file and print CCN per function.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/languages/all"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: analyze_file <path>")
	}
	fi, err := chamele.AnalyzeFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File: %s  NLOC: %d  Functions: %d\n",
		fi.Filename, fi.NLOC, len(fi.Functions))
	for _, fn := range fi.Functions {
		fmt.Printf("  %s  CCN=%d  NLOC=%d  params=%d\n",
			fn.Name, fn.CyclomaticComplexity, fn.NLOC, fn.ParameterCount())
	}
}
