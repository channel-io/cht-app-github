package svc

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
)

type ReleaseSvc struct {
	githubSvc  github.Service
	channelSvc channel.Service
}

func NewReleaseSvc(githubSvc github.Service, channelSvc channel.Service) *ReleaseSvc {
	return &ReleaseSvc{
		githubSvc:  githubSvc,
		channelSvc: channelSvc,
	}
}
func (svc *ReleaseSvc) BuildMessageBlocksFromBody(ctx context.Context, installCtx github.InstallationContext, repository, body string) ([]model.MessageBlock, error) {
	group, err := svc.githubSvc.FindReleaseGroup(ctx, installCtx, repository)
	if err != nil {
		return nil, err
	}

	return svc.channelSvc.BuildMessageBlocksFromMarkdown(ctx, group.ChannelID, []byte(body))
}

func (svc *ReleaseSvc) SyncReleaseWithChannelTalk(ctx context.Context, installCtx github.InstallationContext, repository string, message *model.Message) error {
	group, err := svc.githubSvc.FindReleaseGroup(ctx, installCtx, repository)
	if err != nil {
		return err
	}

	_, err = svc.channelSvc.WriteMessage(ctx, group, message)
	if err != nil {
		return err
	}

	return nil
}
