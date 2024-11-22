package route

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/metric/http/route/metric"
	"github.com/channel-io/cht-app-github/internal/http"
)

func Module() fx.Option {
	return fx.Module(
		"route",

		fx.Provide(
			fx.Annotate(
				metric.NewHandler,
				fx.ParamTags(`name:"metric.handler"`),
				fx.As(new(http.Routes)),
				fx.ResultTags(`group:"metric.http.routes"`),
			),
		),
	)
}
