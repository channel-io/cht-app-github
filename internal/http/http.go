package http

import (
	"github.com/gin-gonic/gin"

	"github.com/channel-io/cht-app-github/internal/config"
)

type Router interface {
	gin.IRouter
}

type Routes interface {
	Path() string
	Register(router Router)
}

type Middleware interface {
	Handler() gin.HandlerFunc
}

func Init(
	e *config.Config,
) {
	switch e.Stage {
	case config.StageDevelopment:
		gin.SetMode(gin.DebugMode)
	case config.StageTest:
		gin.SetMode(gin.TestMode)
	case config.StageProduction:
	case config.StageExp:
		gin.SetMode(gin.ReleaseMode)
	}
}
