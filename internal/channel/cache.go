package channel

import (
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/pkg/cache"
)

type ManagerCache interface {
	cache.Cache[map[string]model.Manager]
}
