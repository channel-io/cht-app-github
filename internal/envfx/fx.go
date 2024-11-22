package envfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/config"
)

var Option = fx.Options(
	fx.Invoke(config.Init),
	fx.Provide(config.Load),
)
