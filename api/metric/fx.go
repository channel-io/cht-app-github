package metric

import (
	"errors"
	"log"
	netHttp "net/http"
	"time"

	"golang.org/x/net/context"

	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/metric/http/route"
	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/http"
)

type ServerParams struct {
	fx.In

	Config http.ServerConfig `name:"metric.http.config"`
	Routes []http.Routes     `group:"metric.http.routes"`
}

type ServerResults struct {
	fx.Out

	Server *http.Server `group:"http.servers"`
}

func NewServer(lifeCycle fx.Lifecycle, p ServerParams) (ServerResults, error) {
	server, err := http.NewServer(p.Config, p.Routes, nil)

	lifeCycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func(server *http.Server) {
				log.Println("starting metric server ...")
				if err := server.Run(); err != nil && !errors.Is(err, netHttp.ErrServerClosed) {
					panic(err)
				}
			}(server)
			return nil
		},
		OnStop: func(_ context.Context) error {
			log.Println("stopping metric server ...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown:", err)
				return err
			}
			log.Println("stopped metric server success")
			return nil
		},
	})

	return ServerResults{
		Server: server,
	}, err
}

func HTTPServerModule() fx.Option {
	return fx.Module(
		"api.metric.http_server",

		route.Module(),

		fx.Provide(
			fx.Annotate(
				func(e *config.Config) http.ServerConfig {
					return http.ServerConfig{
						Port: e.API.Metric.HTTP.Port,
					}
				},
				fx.ResultTags(`name:"metric.http.config"`),
			),
		),

		fx.Provide(
			NewServer,
		),
	)
}
