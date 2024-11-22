package svc

import (
	"context"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
)

type StatusSvc struct {
	githubSvc  github.Service
	channelSvc channel.Service
}

func NewStatusSvc(githubSvc github.Service, channelSvc channel.Service) *StatusSvc {
	return &StatusSvc{
		githubSvc:  githubSvc,
		channelSvc: channelSvc,
	}
}

func (svc *StatusSvc) SyncCommitStatusWithChannelTalk(
	ctx context.Context,
	installCtx github.InstallationContext,
	repository string,
	commitSHA string,
	message *model.Message,
) (err error) {
	// NOTE : filter closed 를 하는 이유는, 해당 pr이 merge 된 이후에만 status 를 메시지로 작성하기 위함입니다.
	pullRequests, err := svc.githubSvc.ListPullRequestNumberByCommitSHA(
		ctx,
		installCtx,
		repository,
		commitSHA,
		github.WithClosedPullRequestFilter(),
		github.WithNotDraftPullRequestFilter(),
		github.WithMergedPullRequestFilter(),
	)
	if err != nil {
		return err
	}
	if len(pullRequests) == 0 {
		return nil
	}

	rootMessageID, err := svc.githubSvc.FindRootMessageID(ctx, installCtx, repository, pullRequests[0].GetNumber(), defaultTryCountFindingComment)
	if err != nil {
		return err
	}

	group, err := svc.githubSvc.FindGroup(ctx, installCtx, repository)
	if err != nil {
		return err
	}

	if rootMessageID != nil {
		return svc.channelSvc.WriteThreadMessage(ctx, group, *rootMessageID, message, false)
	}
	return nil
}
