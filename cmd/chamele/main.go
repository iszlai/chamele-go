// Command chamele analyzes source-code cyclomatic complexity.
//
// It is a pure-Go port of lizard (https://github.com/terryyin/lizard),
// targeting behavioural parity with lizard v1.22.1.
package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/ext/all"
	_ "github.com/iszlai/chamele-go/languages/all"
	"github.com/iszlai/chamele-go/output"
)

// errWarningGate is returned by run() when the -i (ignore-warnings) gate
// is exceeded. main() translates it to exit code 1 without printing.
var errWarningGate = errors.New("warning threshold exceeded")

func main() {
	err := rootCmd().Execute()
	switch {
	case err == nil:
		return
	case errors.Is(err, errWarningGate):
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

type cliFlags struct {
	languages      []string
	verbose        bool
	ccn            int
	inputFile      string
	outputFile     string
	length         int
	arguments      int
	warningsOnly   bool
	warningMSVS    bool
	ignoreWarnings int
	exclude        []string
	threads        int
	xmlOut         bool
	csvOut         bool
	htmlOut        bool
	modified       bool
	checkstyle     bool
	extensions     []string
	sort           []string
	threshold      []string
	whitelist      string
	version        bool
}

func rootCmd() *cobra.Command {
	f := &cliFlags{}

	cmd := &cobra.Command{
		Use:   "chamele [flags] [PATH or FILE] ...",
		Short: "A code complexity analyzer — Go port of lizard",
		Long: `chamele measures NLOC, cyclomatic complexity (CCN), token count,
parameter count and function length for 27+ programming languages
without needing imports or headers resolved.`,
		Args:          cobra.ArbitraryArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, f)
		},
	}

	fl := cmd.Flags()
	fl.StringSliceVarP(&f.languages, "languages", "l", nil, "limit analysis to these languages (repeatable)")
	fl.BoolVarP(&f.verbose, "verbose", "V", false, "show long function names")
	fl.IntVarP(&f.ccn, "ccn", "C", 15, "CCN warning threshold")
	fl.StringVarP(&f.inputFile, "input-file", "f", "", "read paths from file (one per line)")
	fl.StringVarP(&f.outputFile, "output-file", "o", "", "write output to file (format inferred from extension)")
	fl.IntVarP(&f.length, "length", "L", 1000, "function length warning threshold")
	fl.IntVarP(&f.arguments, "arguments", "a", 100, "parameter count warning threshold")
	fl.BoolVarP(&f.warningsOnly, "warnings-only", "w", false, "output clang-style warnings only")
	fl.BoolVar(&f.warningMSVS, "warning-msvs", false, "output MSVS-style warnings only")
	fl.IntVarP(&f.ignoreWarnings, "ignore-warnings", "i", -1, "exit(1) if warnings exceed this count (-1 = never)")
	fl.StringArrayVarP(&f.exclude, "exclude", "x", nil, "fnmatch exclude patterns (repeatable)")
	fl.IntVarP(&f.threads, "threads", "t", runtime.NumCPU(), "number of parallel workers")
	fl.BoolVarP(&f.xmlOut, "xml", "X", false, "XML output (cppncss-compatible)")
	fl.BoolVar(&f.csvOut, "csv", false, "CSV output")
	fl.BoolVarP(&f.htmlOut, "html", "H", false, "HTML output")
	fl.BoolVarP(&f.modified, "modified", "m", false, "use modified CCN (switch/case counts as 1)")
	fl.BoolVar(&f.checkstyle, "checkstyle", false, "Checkstyle XML output")
	fl.StringArrayVarP(&f.extensions, "extension", "E", nil, "enable extension by name (repeatable)")
	fl.StringArrayVarP(&f.sort, "sort", "s", nil, "sort warnings by field (repeatable)")
	fl.StringArrayVarP(&f.threshold, "threshold", "T", nil, "set metric threshold: field=value (repeatable)")
	fl.StringVarP(&f.whitelist, "whitelist", "W", "whitelizard.txt", "whitelist file path")
	fl.BoolVar(&f.version, "version", false, "print version and exit")

	return cmd
}

func run(args []string, f *cliFlags) error {
	if f.version {
		fmt.Println("chamele", chamele.Version)
		return nil
	}

	// Collect paths from args + optional input file.
	paths := args
	if f.inputFile != "" {
		extra, err := readLines(f.inputFile)
		if err != nil {
			return fmt.Errorf("reading --input-file: %w", err)
		}
		paths = append(paths, extra...)
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Build options. The whitelist is an output-stage concern and is passed
	// directly to TabularOptions further down.
	opts := []chamele.Option{
		chamele.WithThreads(f.threads),
		chamele.WithExclude(f.exclude...),
	}
	if len(f.languages) > 0 {
		opts = append(opts, chamele.WithLanguages(f.languages...))
	}

	// Run analysis.
	files, exts, err := chamele.AnalyzeWithExtensions(paths, opts...)
	if err != nil {
		return err
	}

	// Build thresholds.
	thresholds := []chamele.Threshold{
		{Metric: "cyclomatic_complexity", Limit: f.ccn},
		{Metric: "length", Limit: f.length},
		{Metric: "parameter_count", Limit: f.arguments},
	}
	for _, spec := range f.threshold {
		parts := strings.SplitN(spec, "=", 2)
		if len(parts) == 2 {
			var val int
			if _, err := fmt.Sscanf(parts[1], "%d", &val); err != nil {
				continue
			}
			thresholds = append(thresholds, chamele.Threshold{Metric: parts[0], Limit: val})
		}
	}

	// Choose output destination.
	out := os.Stdout
	if f.outputFile != "" {
		fh, err := os.Create(f.outputFile)
		if err != nil {
			return fmt.Errorf("opening output file: %w", err)
		}
		defer func() { _ = fh.Close() }()
		out = fh
		// Infer format from extension if no explicit flag is set.
		switch {
		case strings.HasSuffix(f.outputFile, ".xml") && !f.xmlOut:
			f.xmlOut = true
		case strings.HasSuffix(f.outputFile, ".csv") && !f.csvOut:
			f.csvOut = true
		case strings.HasSuffix(f.outputFile, ".html") && !f.htmlOut:
			f.htmlOut = true
		}
	}

	// Emit output.
	var warnCount int
	switch {
	case f.xmlOut:
		output.PrintXML(out, files, f.verbose)
	case f.csvOut:
		output.PrintCSV(out, files, f.verbose)
	case f.htmlOut:
		output.PrintHTML(out, files)
	case f.checkstyle:
		output.PrintCheckstyle(out, files, thresholds)
	case f.warningsOnly:
		warnCount = output.PrintClangWarnings(out, files, thresholds)
	case f.warningMSVS:
		warnCount = output.PrintMSVSWarnings(out, files, thresholds)
	default:
		warnCount = output.PrintTabular(out, files, output.TabularOptions{
			Thresholds: thresholds,
			Sort:       f.sort,
			Whitelist:  f.whitelist,
			Verbose:    f.verbose,
		})
	}

	// Per-extension summaries (boolcount rate, duplicate-block listing, …).
	// Skipped for machine-readable formats so we don't pollute their syntax.
	if !f.xmlOut && !f.csvOut && !f.htmlOut && !f.checkstyle {
		_ = chamele.RunPrinters(out, exts)
	}

	// Exit-code gate: -i flag. Return a sentinel error rather than calling
	// os.Exit so test harnesses can intercept it.
	if f.ignoreWarnings >= 0 && warnCount > f.ignoreWarnings {
		return errWarningGate
	}
	return nil
}

func readLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
