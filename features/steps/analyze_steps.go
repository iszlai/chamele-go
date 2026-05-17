package steps

import (
	"fmt"

	"github.com/cucumber/godog"
	"github.com/iszlai/chamele-go/chamele"
	_ "github.com/iszlai/chamele-go/languages/all"
)

func registerAnalyzeSteps(sc *godog.ScenarioContext, w *World) {
	sc.Step(`^I analyze it$`, func() error {
		path, err := writeTempFile(w.SourceCode, langExt(w.Lang))
		if err != nil {
			return err
		}
		fi, err := chamele.AnalyzeFile(path)
		if err != nil {
			return err
		}
		if fi != nil {
			w.Results = []chamele.FileInformation{*fi}
		}
		return nil
	})

	sc.Step(`^I analyze it with options:$`, func(table *godog.Table) error {
		var opts []chamele.Option
		for _, row := range table.Rows[1:] {
			key := row.Cells[0].Value
			val := row.Cells[1].Value
			switch key {
			case "languages":
				opts = append(opts, chamele.WithLanguages(val))
			}
		}
		path, err := writeTempFile(w.SourceCode, langExt(w.Lang))
		if err != nil {
			return err
		}
		fi, err := chamele.AnalyzeFile(path, opts...)
		if err != nil {
			return err
		}
		if fi != nil {
			w.Results = []chamele.FileInformation{*fi}
		}
		return nil
	})
}

// ensure import used
var _ = fmt.Sprintf
