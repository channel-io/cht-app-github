package route

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/public/http/route/function"
	"github.com/channel-io/cht-app-github/api/public/http/route/hook"
	"github.com/channel-io/cht-app-github/api/public/http/route/ping"
	"github.com/channel-io/cht-app-github/api/public/http/route/swagger"
	"github.com/channel-io/cht-app-github/api/public/http/route/version"
	"github.com/channel-io/cht-app-github/internal/http"
)

func Module() fx.Option {
	return fx.Module(
		"route",

		fx.Provide(
			route(ping.NewHandler),
			route(swagger.NewHandler),
			route(version.NewHandler),
			route(hook.NewHandler),
			route(function.NewHandler),
		),
	)
}

func route(fn interface{}) interface{} {
	return fx.Annotate(
		fn,
		fx.As(new(http.Routes)),
		fx.ResultTags(`group:"public.http.routes"`),
	)
}
