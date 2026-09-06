package bdd

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
)

// TestFeatures executa os cenários Gherkin em test/bdd/features contra a API
// real (localhost:8080). Requer que os containers estejam em execução:
// docker compose up -d
// É pulado por padrão (inclusive em "go test ./..." e no test runner do
// VS Code) a menos que a variável de ambiente E2E=1 esteja definida.
func TestFeatures(t *testing.T) {
	if os.Getenv("E2E") != "1" {
		t.Skip(`skipping e2e BDD tests: set E2E=1 and run "docker compose up -d" first`)
	}

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}
