package tool

import (
	"context"
	"log"
	"net/http/httptest"
	"time"

	"go.uber.org/fx"
)

type IntegratedTestSuite struct {
	opts  []fx.Option
	tests []interface{}
}

func NewIntegratedTestSuite() *IntegratedTestSuite {
	return &IntegratedTestSuite{}
}

// Specifies the target module to be tested.
func (s *IntegratedTestSuite) Target(module fx.Option) *IntegratedTestSuite {
	s.opts = append(s.opts, module)
	return s
}

// Mock allows test suites to specify which dependencies to be mocked.
// This is useful for testing a specific module in isolation.
//
// Use fx.Decorate to override the dependency with a mocked implementation.
func (s *IntegratedTestSuite) Mock(mock fx.Option) *IntegratedTestSuite {
	s.opts = append(s.opts, mock)
	return s
}

// Adds a test function to the test suite.
func (s *IntegratedTestSuite) Test(test interface{}) *IntegratedTestSuite {
	s.tests = append(s.tests, test)
	return s
}

// Runs the test suite.
func (s *IntegratedTestSuite) Run() {
	app := fx.New(
		fx.Options(s.opts...),
		integratedTestModule(),
		fx.Invoke(s.tests...),
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		log.Fatal(err)
	}
}

// Common modules provided for integrated testing.
func integratedTestModule() fx.Option {
	return fx.Module(
		"integrated_test",
		fx.Provide(httptest.NewRecorder),
	)
}
