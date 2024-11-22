package function

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
)

const (
	ErrTypeMethodNotFound      = "MethodNotFound"
	ErrTypeInvalidParams       = "InvalidParams"
	ErrTypeUnProcessableEntity = "UnProcessableEntity"
)

type ErrType string

type JsonFunctionDelegator struct {
	registry HandlerRegistry
}

func NewJsonFunctionDelegator(todoFunc *TODOFunction) *JsonFunctionDelegator {
	registry := make(HandlerRegistry)
	todoFunc.Register(registry)
	return &JsonFunctionDelegator{registry: registry}
}

func (h *JsonFunctionDelegator) Delegate(ctx context.Context, req *appstore.JsonFunctionRequest) *appstore.JsonFunctionResponse {
	handlerFn, exists := h.registry[req.Method]
	if !exists {
		return ErrResponse(ErrTypeMethodNotFound, fmt.Sprintf("method %s not found", req.Method))
	}

	b, err := json.Marshal(req.Params)
	if err != nil {
		return ErrResponse(ErrTypeInvalidParams, err.Error())
	}

	if err = handlerFn(ctx, b, req.Context); err != nil {
		return ErrResponse(ErrTypeUnProcessableEntity, err.Error())
	}

	return SuccessResponse()
}

func ErrResponse(errType ErrType, message string) *appstore.JsonFunctionResponse {
	return &appstore.JsonFunctionResponse{
		Error: &appstore.Error{
			Type:    string(errType),
			Message: message,
		},
	}
}

func ResultResponse(res json.RawMessage) *appstore.JsonFunctionResponse {
	return &appstore.JsonFunctionResponse{
		Result: res,
	}
}

func SuccessResponse() *appstore.JsonFunctionResponse {
	result, _ := json.Marshal(map[string]any{
		"success": true,
	})
	return &appstore.JsonFunctionResponse{
		Result: result,
	}
}
