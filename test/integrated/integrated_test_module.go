package integrated

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/channelfx"
	"github.com/channel-io/cht-app-github/internal/envfx"
	"github.com/channel-io/cht-app-github/internal/eventfx"
	"github.com/channel-io/cht-app-github/internal/functionfx"
	"github.com/channel-io/cht-app-github/internal/githubfx"
	"github.com/channel-io/cht-app-github/internal/httpfx"
	"github.com/channel-io/cht-app-github/internal/loggerfx"
)

func integratedTestModule() fx.Option {
	return fx.Module(
		"test.integrated",
		channelfx.Option,
		envfx.Option,
		httpfx.Option,
		loggerfx.Option,
		eventfx.Option,
		githubfx.Module(),
		functionfx.Module(),
	)
}
