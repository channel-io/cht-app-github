package svc

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
)

func NewCommonSvc(githubSvc github.Service, channelSvc channel.Service) *CommonSvc {
	return &CommonSvc{
		githubSvc:  githubSvc,
		channelSvc: channelSvc,
	}
}

type CommonSvc struct {
	githubSvc  github.Service
	channelSvc channel.Service
}

func (u *CommonSvc) BuildManagerMentionTextByGithubUsername(ctx context.Context, installCtx github.InstallationContext, repository, username string) (string, error) {
	group, err := u.githubSvc.FindGroup(ctx, installCtx, repository)
	if err != nil {
		return "", err
	}
	manager, err := u.channelSvc.FindManagerByGitHubMentionUsername(ctx, group.ChannelID, username)
	if err != nil {
		return "", err
	}
	if manager != nil {
		return model.Mention(model.MentionTypeManager, manager.ID, manager.Name), nil
	}
	return username, nil
}

func (u *CommonSvc) FindManagerNameByGithubUsername(ctx context.Context, installCtx github.InstallationContext, repository, username string) (string, error) {
	group, err := u.githubSvc.FindGroup(ctx, installCtx, repository)
	if err != nil {
		return "", err
	}
	manager, err := u.channelSvc.FindManagerByGitHubMentionUsername(ctx, group.ChannelID, username)
	if err != nil {
		return "", err
	}
	if manager != nil {
		return manager.Name, nil
	}
	return username, nil
}

func (u *CommonSvc) IgnoreBot(ctx context.Context, orgLogin, repoName string) bool {
	// TODO: config by custom property
	if orgLogin == "channel-io" && repoName == "k8s" {
		return true
	}
	return false
}
