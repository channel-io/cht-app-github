package svc

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
)

func NewIssueSvc(githubSvc github.Service, channelSvc channel.Service) *IssueSvc {
	return &IssueSvc{
		githubSvc:  githubSvc,
		channelSvc: channelSvc,
	}
}

type IssueSvc struct {
	githubSvc  github.Service
	channelSvc channel.Service
}

const defaultTryCountFindingComment = 3

func (u *IssueSvc) SyncIssueWithChannelTalk(
	ctx context.Context,
	installCtx github.InstallationContext,
	repository string,
	issueNumber int,
	message *model.Message,
	opts ...SyncOption,
) (err error) {
	group, err := u.githubSvc.FindGroup(ctx, installCtx, repository)
	if err != nil {
		return err
	}

	c := syncConfig{}
	for _, opt := range opts {
		opt(&c)
	}

	tryFindingCommentCount := defaultTryCountFindingComment
	if c.noRetry {
		tryFindingCommentCount = 0
	}

	rootMessageID, err := u.githubSvc.FindRootMessageID(ctx, installCtx, repository, issueNumber, tryFindingCommentCount)
	if err != nil {
		return err
	}

	if rootMessageID == nil && c.stopWithoutRootMessage {
		return nil
	}

	if rootMessageID != nil {
		return u.channelSvc.WriteThreadMessage(ctx, group, *rootMessageID, message, c.broadcast)
	}

	messageID, err := u.channelSvc.WriteMessage(ctx, group, message)
	if err != nil {
		return err
	}

	url := u.channelSvc.BuildTeamChatURL(group, messageID)
	return u.githubSvc.CreateComment(ctx, installCtx, repository, issueNumber, url)
}

type syncConfig struct {
	broadcast              bool
	noRetry                bool
	stopWithoutRootMessage bool
}

type SyncOption func(*syncConfig)

func WithBroadCasting() SyncOption {
	return func(config *syncConfig) {
		config.broadcast = true
	}
}

func WithoutTryFindingRootMessage() SyncOption {
	return func(config *syncConfig) {
		config.noRetry = true
	}
}

func StopWithoutRootMessage() SyncOption {
	return func(config *syncConfig) {
		config.stopWithoutRootMessage = true
	}
}

func (u *IssueSvc) AddAssigneeToIssue(
	ctx context.Context,
	installCtx github.InstallationContext,
	repository string,
	issueNumber int,
	assignees []string,
) (err error) {
	return u.githubSvc.AddAssigneeToIssue(ctx, installCtx, repository, issueNumber, assignees)
}
