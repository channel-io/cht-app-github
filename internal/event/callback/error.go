package callback

import (
	"github.com/cbrgm/githubevents/githubevents"

	"github.com/channel-io/cht-app-github/internal/logger"
)

func HandleError(
	handler *githubevents.EventHandler,
	logger logger.Logger,
) {
	handler.OnError(func(deliveryID string, eventName string, event interface{}, err error) error {
		logger.Error(err)
		return err
	})
}
