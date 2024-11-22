package hook

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/channel-io/cht-app-github/internal/event"
	libhttp "github.com/channel-io/cht-app-github/internal/http"
)

type Handler struct {
	eventHandler *event.GithubEventHandler
}

func NewHandler(
	eventHandler *event.GithubEventHandler,
) *Handler {
	return &Handler{
		eventHandler: eventHandler,
	}
}

func (h *Handler) Path() string {
	return "/hook/v1"
}

func (h *Handler) Register(router libhttp.Router) {
	router.POST("", h.processEvent)
}

// Ping godoc
//
//	@Summary		Process Event
//	@Description	Process webhook event
//	@Tags			Hook
//	@Produce		plain
//	@Success		200
//	@Router			/hook/v1 [post]
func (h *Handler) processEvent(ctx *gin.Context) {
	if err := h.eventHandler.HandleEventRequest(ctx.Request); err != nil {
		_ = ctx.Error(err)
		return
	}
	ctx.Status(http.StatusOK)
}
