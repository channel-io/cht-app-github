package metricfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/metric"
)

func MetricServerModule() fx.Option {
	return fx.Module(
		"metric",

		fx.Provide(
			fx.Annotate(
				metric.NewRegistry,
				fx.ParamTags(`group:"metric.collector"`),
			),
		),

		fx.Provide(
			fx.Annotate(
				metric.NewHTTPHandler,
				fx.ResultTags(`name:"metric.handler"`),
			),
		),
	)
}
