package functional_test

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
)

func TestFunctional(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping functional suite in -short mode")
	}

	addr := os.Getenv("AUDIT_LOG_FUNCTIONAL_GRPC_ADDR")
	if addr == "" {
		t.Skip("skipping functional suite: AUDIT_LOG_FUNCTIONAL_GRPC_ADDR not set (run `just functional`)")
	}
	testGRPCAddr = addr

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("functional scenarios failed")
	}
}
