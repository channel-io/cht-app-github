package functionfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/function"
)

func Module() fx.Option {
	return fx.Module(
		"function",
		fx.Provide(
			function.NewTODOFunction,
			function.NewJsonFunctionDelegator,
		),
	)
}
