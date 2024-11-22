package swagger

import (
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/channel-io/cht-app-github/internal/http"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Path() string {
	return "/swagger"
}

func (h *Handler) Register(router http.Router) {
	router.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
