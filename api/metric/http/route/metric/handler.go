package metric

import (
	"net/http"

	"github.com/gin-gonic/gin"

	libhttp "github.com/channel-io/cht-app-github/internal/http"
)

type Handler struct {
	metricHandler http.Handler
}

func NewHandler(metricHandler http.Handler) *Handler {
	return &Handler{
		metricHandler: metricHandler,
	}
}

func (h *Handler) Path() string {
	return "/metrics"
}

func (h *Handler) Register(router libhttp.Router) {
	router.GET("", h.Metrics)
}

func (h *Handler) Metrics(ctx *gin.Context) {
	h.metricHandler.ServeHTTP(ctx.Writer, ctx.Request)
}
