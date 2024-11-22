package loggerfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/logger"
)

var Option = fx.Options(
	fx.Provide(
		fx.Annotate(
			logger.NewBasicLogger,
			fx.As(new(logger.Logger)),
		),
	),
)
