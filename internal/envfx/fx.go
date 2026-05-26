package envfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/config"
)

var Option = fx.Options(
	fx.Provide(loadConfig),
)

func loadConfig() (*config.Config, error) {
	config.Init()
	return config.Load()
}
