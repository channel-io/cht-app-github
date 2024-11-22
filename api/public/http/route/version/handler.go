package version

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/channel-io/cht-app-github/internal/config"
	libhttp "github.com/channel-io/cht-app-github/internal/http"
)

type Handler struct {
	env *config.Config
}

func NewHandler(
	env *config.Config,
) *Handler {
	return &Handler{
		env: env,
	}
}

func (h *Handler) Path() string {
	return "/version"
}

func (h *Handler) Register(router libhttp.Router) {
	router.GET("", h.Version)
}

// Version godoc
//
//	@Summary		Version
//	@Description	Retrieves current version of the server.
//	@Tags			Utility
//	@Produce		json
//	@Success		200	{object}	version.ok
//	@Router			/version [get]
func (h *Handler) Version(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, ok{
		Version:   h.env.Build.Version,
		Commit:    h.env.Build.Commit,
		BuildTime: h.env.Build.Time,
	})
}
