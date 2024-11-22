package client

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel/model"
)

type Client interface {
	WriteGroupMessage(ctx context.Context, channelId, groupId string, message *model.Message) (string, error)
	WriteThreadMessage(ctx context.Context, channelId, groupId, rootMessageId string, message *model.Message, broadcast bool) error
	ListManagers(ctx context.Context, channelID string) ([]model.Manager, error)
	GetManager(ctx context.Context, channelID, managerID string) (model.Manager, error)
}
