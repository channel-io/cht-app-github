package httpfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/http"
)

var Option = fx.Option(
	fx.Invoke(http.Init),
)
