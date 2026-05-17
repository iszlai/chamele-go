package steps

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
)

func registerSourceSteps(sc *godog.ScenarioContext, w *World) {
	sc.Step(`^chamele is configured with default options$`, func() error {
		return nil // default — no-op
	})

	sc.Step(`^a Go file containing:$`, func(src *godog.DocString) error {
		w.Lang = "go"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a Python file containing:$`, func(src *godog.DocString) error {
		w.Lang = "py"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a C file containing:$`, func(src *godog.DocString) error {
		w.Lang = "c"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a C\+\+ file containing:$`, func(src *godog.DocString) error {
		w.Lang = "cpp"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a Java file containing:$`, func(src *godog.DocString) error {
		w.Lang = "java"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a JavaScript file containing:$`, func(src *godog.DocString) error {
		w.Lang = "js"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a Rust file containing:$`, func(src *godog.DocString) error {
		w.Lang = "rs"
		w.SourceCode = src.Content
		return nil
	})

	sc.Step(`^a directory with these files:$`, func(table *godog.Table) error {
		dir, err := os.MkdirTemp("", "chamele-bdd-*")
		if err != nil {
			return err
		}
		for _, row := range table.Rows[1:] {
			filename := row.Cells[0].Value
			content := row.Cells[1].Value
			if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
				return err
			}
		}
		w.SourceCode = dir
		return nil
	})
}

func langExt(lang string) string {
	switch lang {
	case "go":
		return "go"
	case "py", "python":
		return "py"
	case "c":
		return "c"
	case "cpp", "c++":
		return "cpp"
	case "java":
		return "java"
	case "js", "javascript":
		return "js"
	case "rs", "rust":
		return "rs"
	default:
		return lang
	}
}

func writeTempFile(content, ext string) (string, error) {
	f, err := os.CreateTemp("", "chamele-bdd-*."+ext)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

// ensure writeTempFile and langExt are used (they're called from analyze_steps)
var _ = fmt.Sprintf
