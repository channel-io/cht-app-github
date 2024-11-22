package githubfx

import (
	"go.uber.org/fx"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/channel-io/cht-app-github/internal/github"
)

type ClientMetricsResult struct {
	fx.Out

	ClientMetrics *github.ClientMetrics
	Collector     prometheus.Collector `group:"metric.collector"`
}

func NewClientMetrics() ClientMetricsResult {
	cm := github.NewClientMetrics()
	return ClientMetricsResult{
		ClientMetrics: cm,
		Collector:     cm,
	}
}

func Module() fx.Option {
	return fx.Module(
		"github",

		fx.Provide(
			NewClientMetrics,
			fx.Annotate(
				github.NewServiceImpl,
				fx.As(new(github.Service)),
			),
		),
	)
}
