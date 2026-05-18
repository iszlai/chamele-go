package steps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cucumber/godog"
)

// langExtMap maps human-readable language names (and a few aliases) to the
// file extension chamele uses to dispatch a reader. Both the parameterised
// step and the legacy per-language steps consult it.
var langExtMap = map[string]string{
	"go":         "go",
	"golang":     "go",
	"python":     "py",
	"py":         "py",
	"c":          "c",
	"c++":        "cpp",
	"cpp":        "cpp",
	"java":       "java",
	"javascript": "js",
	"js":         "js",
	"rust":       "rs",
	"rs":         "rs",
}

func registerSourceSteps(sc *godog.ScenarioContext, w *World) {
	sc.Step(`^chamele is configured with default options$`, func() error {
		return nil // default — no-op
	})

	// Parameterised: `Given a "Go" file containing: """..."""`
	// New scenarios should use this form. The per-language steps below are
	// kept so existing feature files keep working without sweeping renames.
	sc.Step(`^a "([^"]+)" file containing:$`, func(lang string, src *godog.DocString) error {
		w.Lang = canonicalLang(lang)
		w.SourceCode = src.Content
		return nil
	})

	// Legacy per-language steps — convenience aliases for the parameterised
	// form. Each is one line, so the repetition is no longer a maintenance
	// burden.
	for _, lang := range []string{"Go", "Python", "C", "C\\+\\+", "Java", "JavaScript", "Rust"} {
		l := canonicalLang(strings.ReplaceAll(lang, "\\", ""))
		sc.Step(`^a `+lang+` file containing:$`, func(src *godog.DocString) error {
			w.Lang = l
			w.SourceCode = src.Content
			return nil
		})
	}

	sc.Step(`^a directory with these files:$`, func(table *godog.Table) error {
		dir, err := os.MkdirTemp("", "chamele-bdd-*")
		if err != nil {
			return err
		}
		for _, row := range table.Rows[1:] {
			filename := row.Cells[0].Value
			content := row.Cells[1].Value
			if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
				return err
			}
		}
		w.SourceCode = dir
		return nil
	})
}

func canonicalLang(s string) string {
	if ext, ok := langExtMap[strings.ToLower(s)]; ok {
		return ext
	}
	return strings.ToLower(s)
}

func langExt(lang string) string {
	if ext, ok := langExtMap[strings.ToLower(lang)]; ok {
		return ext
	}
	return lang
}

func writeTempFile(content, ext string) (string, error) {
	f, err := os.CreateTemp("", "chamele-bdd-*."+ext)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		return "", err
	}
	_ = f.Close()
	return f.Name(), nil
}
