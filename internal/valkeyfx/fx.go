package valkeyfx

import (
	"context"

	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/logger"
	"github.com/channel-io/cht-app-github/internal/valkey"
)

var Module = fx.Module(
	"valkey",
	fx.Provide(
		provideRepository,
	),
	fx.Invoke(lifecycleHook),
)

func provideRepository(lc fx.Lifecycle, cfg *config.Config, log logger.Logger) (*valkey.Repository, error) {
	client, err := valkey.NewClient(context.Background(), cfg, log)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return valkey.NewRepository(nil), nil
	}

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return client.Close()
		},
	})

	return valkey.NewRepository(client), nil
}

func lifecycleHook(_ *valkey.Repository) {}
