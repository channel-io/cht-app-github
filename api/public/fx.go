package public

import (
	"errors"
	"log"
	netHttp "net/http"
	"time"

	"golang.org/x/net/context"

	_ "github.com/channel-io/cht-app-github/api/public/http/docs"

	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/public/http/route"
	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/http"
)

type ServerParams struct {
	fx.In

	Config      http.ServerConfig `name:"public.http.config"`
	Routes      []http.Routes     `group:"public.http.routes"`
	Middlewares []http.Middleware `group:"public.http.middlewares"`
}

type ServerResults struct {
	fx.Out

	Server *http.Server `group:"http.servers"`
}

func NewServer(lifeCycle fx.Lifecycle, p ServerParams) (ServerResults, error) {
	server, err := http.NewServer(p.Config, p.Routes, p.Middlewares)

	lifeCycle.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func(server *http.Server) {
				log.Println("starting server ...")
				if err := server.Run(); err != nil && !errors.Is(err, netHttp.ErrServerClosed) {
					panic(err)
				}
			}(server)
			return nil
		},
		OnStop: func(_ context.Context) error {
			log.Println("stopping server ...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown:", err)
				return err
			}
			log.Println("stopped server success")
			return nil
		},
	})

	return ServerResults{
		Server: server,
	}, err
}

func HTTPServerModule() fx.Option {
	return fx.Module(
		"api.public.http_server",

		route.Module(),

		fx.Provide(
			fx.Annotate(
				func(e *config.Config) http.ServerConfig {
					return http.ServerConfig{
						Port: e.API.Public.HTTP.Port,
					}
				},
				fx.ResultTags(`name:"public.http.config"`),
			),
		),

		fx.Provide(
			NewServer,
		),
	)
}

func HTTPServerTestModule() fx.Option {
	return fx.Module(
		"test.api.public.http_server",

		route.Module(),

		fx.Provide(
			fx.Annotate(
				func(e *config.Config) http.ServerConfig {
					return http.ServerConfig{
						Port: e.API.Public.HTTP.Port,
					}
				},
				fx.ResultTags(`name:"public.http.config"`),
			),
		),

		fx.Provide(
			fx.Annotate(
				http.NewServer,
				fx.ParamTags(
					`name:"public.http.config"`,
					`group:"public.http.routes"`,
					`group:"public.http.middlewares"`,
				),
			),
		),
	)
}
