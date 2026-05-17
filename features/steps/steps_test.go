package steps

import (
	"context"
	"os"
	"testing"

	"github.com/cucumber/godog"
)

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../"},
			TestingT: t,
			Tags:     os.Getenv("GODOG_TAGS"),
		},
	}
	if suite.Run() != 0 {
		t.Fatal("BDD suite failed")
	}
}

func InitializeScenario(sc *godog.ScenarioContext) {
	w := &World{}
	sc.Before(func(_ context.Context, _ *godog.Scenario) (context.Context, error) {
		w.Reset()
		return context.Background(), nil
	})
}
