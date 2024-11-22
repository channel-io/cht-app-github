package channelfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/channel"
	channelclient "github.com/channel-io/cht-app-github/internal/channel/client"
	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
	"github.com/channel-io/cht-app-github/internal/config"
)

var Option = fx.Options(
	fx.Provide(
		fx.Annotate(
			channelclient.NewNativeFunction,
			fx.As(new(channelclient.Client)),
		),
		channel.NewServiceImpl,
	),

	// deps
	fx.Provide(
		fx.Annotate(
			appstore.NewClient,
			fx.ParamTags(`name:"appstore.baseurl"`, `name:"appstore.secret"`),
		),

		fx.Annotate(
			func(conf *config.Config) (string, string) {
				return conf.ChannelTalk.AppStore.BaseUrl, conf.ChannelTalk.App.Secret
			},
			fx.ResultTags(`name:"appstore.baseurl"`, `name:"appstore.secret"`),
		),
	),
)
