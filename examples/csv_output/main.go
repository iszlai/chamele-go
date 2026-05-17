// Example: walk a directory and write CSV to stdout.
package main

import (
	"log"
	"os"

	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/languages/all"
	"github.com/iszlai/chamele-go/output"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	files, err := chamele.Analyze([]string{dir})
	if err != nil {
		log.Fatal(err)
	}
	output.PrintCSV(os.Stdout, files, true)
}
