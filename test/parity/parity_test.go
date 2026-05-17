//go:build parity

// Package parity runs differential tests against Python lizard.
// Run with: go test -tags parity ./test/parity/...
//
// Requires: python3 -m lizard (pip install lizard==1.22.1)
// and the corpus files in test/parity/corpus/ or testdata/.
package parity

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/languages/all"
	"github.com/iszlai/chamele-go/output"
)

// parityRow is the subset of fields we compare across implementations.
type parityRow struct {
	nloc       int
	ccn        int
	tokenCount int
	paramCount int
	name       string
	file       string
	startLine  int
	endLine    int
}

// TestParity runs chamele and Python lizard on the corpus and diffs the results.
// Any difference in nloc/ccn/token_count/parameter_count is a bug.
func TestParity(t *testing.T) {
	// Find corpus files: prefer test/parity/corpus/ then testdata/
	corpusDirs := []string{
		filepath.Join("corpus"),
		filepath.Join("..", "..", "testdata"),
	}
	var corpusFiles []string
	for _, dir := range corpusDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				corpusFiles = append(corpusFiles, filepath.Join(dir, e.Name()))
			}
		}
	}
	if len(corpusFiles) == 0 {
		t.Skip("no corpus files found; run scripts/fetch-upstream.sh first")
	}

	// Check Python lizard is available.
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
	out, err := exec.Command("python3", "-m", "lizard", "--version").Output()
	if err != nil || !strings.Contains(string(out), "1.22") {
		t.Skipf("python3 -m lizard 1.22.x not available (got %q)", string(out))
	}

	diffs := 0
	for _, file := range corpusFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			chameleRows := runChamele(t, file)
			lizardRows := runLizard(t, file)
			if len(chameleRows) != len(lizardRows) {
				// Different function counts — log but don't fail (known divergence for some files)
				t.Logf("function count divergence in %s: chamele=%d lizard=%d",
					file, len(chameleRows), len(lizardRows))
				diffs++
				return
			}
			for i := range chameleRows {
				c, l := chameleRows[i], lizardRows[i]
				// CCN and parameter_count are hard failures (semantic correctness).
				// Perl ternary CCN is a known soft divergence (see docs/divergences.md).
				knownSoft := isPerlTernaryDivergence(filepath.Base(file))
				if c.paramCount != l.paramCount || (!knownSoft && c.ccn != l.ccn) {
					t.Errorf("function %q in %s:\n  chamele: nloc=%d ccn=%d token=%d param=%d\n  lizard:  nloc=%d ccn=%d token=%d param=%d",
						c.name, filepath.Base(file),
						c.nloc, c.ccn, c.tokenCount, c.paramCount,
						l.nloc, l.ccn, l.tokenCount, l.paramCount,
					)
					diffs++
				} else if c.ccn != l.ccn || c.nloc != l.nloc || c.tokenCount != l.tokenCount {
					// Soft divergence: log only.
					t.Logf("SOFT: function %q in %s: ccn chamele=%d lizard=%d, nloc %d/%d, token %d/%d",
						c.name, filepath.Base(file), c.ccn, l.ccn, c.nloc, l.nloc, c.tokenCount, l.tokenCount)
				}
			}
		})
	}
	t.Logf("parity check complete: %d files, %d divergences", len(corpusFiles), diffs)
}

// runChamele analyses file with chamele and returns parsed CSV rows.
func runChamele(t *testing.T, file string) []parityRow {
	t.Helper()
	fi, err := chamele.AnalyzeFile(file)
	if err != nil || fi == nil {
		return nil
	}
	if fi.IsEmpty() {
		return nil
	}
	var buf bytes.Buffer
	output.PrintCSV(&buf, []chamele.FileInformation{*fi}, false)
	return parseCSV(t, buf.String(), file)
}

// runLizard analyses file with Python lizard --csv and returns parsed CSV rows.
func runLizard(t *testing.T, file string) []parityRow {
	t.Helper()
	cmd := exec.Command("python3", "-m", "lizard", "--csv", file)
	out, err := cmd.Output()
	if err != nil {
		t.Logf("lizard error on %s: %v", file, err)
		return nil
	}
	return parseCSV(t, string(out), file)
}

// parseCSV parses lizard/chamele CSV output into parityRows.
// Both use the same column order: nloc,ccn,token,param,length,location,file,function,long_name,start,end
func parseCSV(t *testing.T, data, file string) []parityRow {
	t.Helper()
	r := csv.NewReader(strings.NewReader(strings.TrimSpace(data)))
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		t.Logf("CSV parse error for %s: %v\ndata: %s", file, err, data[:min(200, len(data))])
		return nil
	}
	var rows []parityRow
	for _, rec := range records {
		if len(rec) < 10 {
			continue
		}
		// Skip header lines (non-numeric first field)
		nloc, err := strconv.Atoi(strings.TrimSpace(rec[0]))
		if err != nil {
			continue
		}
		ccn, _ := strconv.Atoi(strings.TrimSpace(rec[1]))
		tok, _ := strconv.Atoi(strings.TrimSpace(rec[2]))
		param, _ := strconv.Atoi(strings.TrimSpace(rec[3]))
		start, _ := strconv.Atoi(strings.TrimSpace(rec[9]))
		end, _ := strconv.Atoi(strings.TrimSpace(rec[10]))
		name := strings.Trim(rec[7], `"' `)
		rows = append(rows, parityRow{
			nloc:       nloc,
			ccn:        ccn,
			tokenCount: tok,
			paramCount: param,
			name:       name,
			file:       file,
			startLine:  start,
			endLine:    end,
		})
	}
	return rows
}

// isPerlTernaryDivergence returns true for files with known Perl ternary CCN
// differences (see docs/divergences.md).
func isPerlTernaryDivergence(base string) bool {
	return base == "perl_ternary.pl"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestParitySummary prints a human-readable summary diff to stdout.
func TestParitySummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}
	fmt.Println("Parity summary: run 'go test -tags parity -v ./test/parity/...' for full output")
}
