package event

import (
	"github.com/cbrgm/githubevents/githubevents"

	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/internal/event/callback"
	"github.com/channel-io/cht-app-github/internal/logger"
)

type GithubEventHandler struct {
	*githubevents.EventHandler
}

type EventCallback interface {
	Register(handler *githubevents.EventHandler)
}

func NewGithubEventHandler(
	config *config.Config,
	logger logger.Logger,
	callbacks []EventCallback,
) *GithubEventHandler {
	h := githubevents.New(config.Github.App.WebhookSecret)
	callback.HandleError(h, logger)

	for _, callback := range callbacks {
		callback.Register(h)
	}

	return &GithubEventHandler{
		EventHandler: h,
	}
}
