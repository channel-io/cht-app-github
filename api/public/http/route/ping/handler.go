package ping

import (
	"net/http"

	"github.com/gin-gonic/gin"

	libhttp "github.com/channel-io/cht-app-github/internal/http"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Path() string {
	return "/ping"
}

func (h *Handler) Register(router libhttp.Router) {
	router.GET("", h.Ping)
}

// Ping godoc
//
//	@Summary		Ping
//	@Description	Ping
//	@Tags			Utility
//	@Produce		plain
//	@Success		200	{string}	string	"pong"
//	@Router			/ping [get]
func (h *Handler) Ping(ctx *gin.Context) {
	ctx.String(http.StatusOK, "pong")
}
