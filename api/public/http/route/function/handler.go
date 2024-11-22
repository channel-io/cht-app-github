package function

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
	"github.com/channel-io/cht-app-github/internal/function"
	libhttp "github.com/channel-io/cht-app-github/internal/http"
	"github.com/channel-io/cht-app-github/internal/logger"
)

type Handler struct {
	functionDelegator *function.JsonFunctionDelegator
	logger            logger.Logger
}

func NewHandler(functionDelegator *function.JsonFunctionDelegator, logger logger.Logger) *Handler {
	return &Handler{
		functionDelegator: functionDelegator,
		logger:            logger,
	}
}

func (h *Handler) Path() string {
	return "/function"
}

func (h *Handler) Register(router libhttp.Router) {
	router.PUT("", h.handleFunction)
}

// function godoc
//
// @Summary	handle gif function
// @Tags		function

// @Success	200 {object} any "response for method"
// @Failure	422

// @Router		/function [put]
func (h *Handler) handleFunction(ctx *gin.Context) {
	var req appstore.JsonFunctionRequest
	if err := ctx.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res := h.functionDelegator.Delegate(ctx, &req)
	if res.Error != nil {
		h.logger.Error(res.Error)
		ctx.JSON(http.StatusOK, res)
		return
	}
	ctx.JSON(http.StatusOK, res)
}
