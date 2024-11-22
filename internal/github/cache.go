package github

import (
	"github.com/channel-io/cht-app-github/pkg/cache"
)

type Cache[T any] interface {
	cache.Cache[T]
}
