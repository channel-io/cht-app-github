package main

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/metric"
	"github.com/channel-io/cht-app-github/api/public"
	"github.com/channel-io/cht-app-github/internal/channelfx"
	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/envfx"
	"github.com/channel-io/cht-app-github/internal/eventfx"
	"github.com/channel-io/cht-app-github/internal/functionfx"
	"github.com/channel-io/cht-app-github/internal/githubfx"
	"github.com/channel-io/cht-app-github/internal/http"
	"github.com/channel-io/cht-app-github/internal/httpfx"
	"github.com/channel-io/cht-app-github/internal/logger"
	"github.com/channel-io/cht-app-github/internal/loggerfx"
	"github.com/channel-io/cht-app-github/internal/metricfx"
)

const (
	appName = "cht-app-github"
)

// @title GO HTTP server
func main() {
	fx.New(
		public.HTTPServerModule(),
		metric.HTTPServerModule(),
		internalModule(),

		fx.NopLogger,
		fx.Invoke(printLog),

		fx.Invoke(
			fx.Annotate(
				func(_ []*http.Server) error {
					return nil
				},
				fx.ParamTags(`group:"http.servers"`),
			),
		),
	).Run()
}

func internalModule() fx.Option {
	return fx.Module(
		"internal",
		channelfx.Option,
		envfx.Option,
		httpfx.Option,
		loggerfx.Option,
		eventfx.Option,
		metricfx.MetricServerModule(),
		functionfx.Module(),
		githubfx.Module(),
	)
}

func printLog(
	env *config.Config,
	logger logger.Logger,
) {
	logger.Infow("Running application", "name", appName, "stage", env.Stage, "version", env.Build.Version)
}
