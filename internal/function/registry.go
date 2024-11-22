package function

import (
	"context"
	"encoding/json"

	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
)

type HandlerRegistry map[string]HandlerFunc

type HandlerFunc func(ctx context.Context, params json.RawMessage, fnCtx appstore.Context) error

func (r HandlerRegistry) Register(method string, handlerFn HandlerFunc) {
	r[method] = handlerFn
}

type HandlerRegistrant interface {
	Register(registry HandlerRegistry)
}
