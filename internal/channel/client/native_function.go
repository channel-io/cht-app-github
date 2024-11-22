package client

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/config"
)

type NativeFunction struct {
	client  *appstore.Client
	botName string
}

func NewNativeFunction(conf *config.Config, client *appstore.Client) *NativeFunction {
	return &NativeFunction{
		client:  client,
		botName: conf.ChannelTalk.BotName,
	}
}

func (s *NativeFunction) WriteGroupMessage(ctx context.Context, channelID, groupID string, message *model.Message) (string, error) {
	res, err := s.client.WriteGroupMessage(ctx, &appstore.WriteGroupMessageRequest{
		ChannelID: channelID,
		GroupID:   groupID,
		DTO: appstore.GroupMessageDTO{
			Blocks:  message.Blocks,
			BotName: s.botName,
		},
	})
	if err != nil {
		return "", err
	}

	return res.Message.ID, nil
}

func (s *NativeFunction) WriteThreadMessage(ctx context.Context, channelID, groupID, rootMessageID string, message *model.Message, broadcast bool) error {
	_, err := s.client.WriteGroupMessage(ctx, &appstore.WriteGroupMessageRequest{
		ChannelID:     channelID,
		GroupID:       groupID,
		RootMessageID: rootMessageID,
		Broadcast:     broadcast,
		DTO: appstore.GroupMessageDTO{
			Blocks:  message.Blocks,
			BotName: s.botName,
		},
	})
	return err
}

const pageSize = 100

func (s *NativeFunction) ListManagers(ctx context.Context, channelID string) ([]model.Manager, error) {
	next := ""
	managers := make([]model.Manager, 0, pageSize)

	req := appstore.SearchManagersRequest{
		ChannelID: channelID,
		Pagination: appstore.Pagination{
			SortOrder: appstore.SortOrderAsc,
			Limit:     pageSize,
		},
	}

	for {
		if next != "" {
			req.Pagination.Since = next
		}

		res, err := s.client.SearchManagers(ctx, &req)
		if err != nil {
			return nil, err
		}

		for _, m := range res.Managers {
			managers = append(managers, m.ToModel())
		}

		if len(res.Managers) < pageSize {
			break
		}
		next = res.Next
	}

	return managers, nil
}

func (s *NativeFunction) GetManager(ctx context.Context, channelID, managerID string) (model.Manager, error) {
	req := appstore.GetManagerRequest{
		ChannelID: channelID,
		ManagerID: managerID,
	}

	res, err := s.client.GetManager(ctx, &req)
	if err != nil {
		return model.Manager{}, err
	}

	return res.Manager.ToModel(), nil
}
