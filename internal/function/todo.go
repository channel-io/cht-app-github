package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/channel/client/appstore"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
	"github.com/channel-io/cht-app-github/internal/logger"
)

type TODOFunction struct {
	logger     logger.Logger
	githubSvc  github.Service
	channelSvc channel.Service
}

func NewTODOFunction(logger logger.Logger, githubSvc github.Service, channelSvc channel.Service) *TODOFunction {
	return &TODOFunction{
		logger:     logger,
		githubSvc:  githubSvc,
		channelSvc: channelSvc,
	}
}

func (f *TODOFunction) Register(registry HandlerRegistry) {
	registry.Register("githubTODO", f.githubTODO)
}

func (f *TODOFunction) githubTODO(
	ctx context.Context,
	params json.RawMessage,
	fnCtx appstore.Context,
) error {
	if fnCtx.Caller.Type != appstore.ManagerCallerType {
		f.logger.Errorw("caller type %s is not manager ", fnCtx.Caller.Type)
		return errors.Errorf("caller type %s is not manager ", fnCtx.Caller.Type)
	}

	var fnParams appstore.CommandParams
	if err := json.Unmarshal(params, &fnParams); err != nil {
		return err
	}
	todoParams := TODOParams{}
	if err := json.Unmarshal(fnParams.Input, &todoParams); err != nil {
		return err
	}

	manager, err := f.channelSvc.FetchManagerByManagerID(ctx, fnCtx.Channel.ID, fnCtx.Caller.ID)
	if err != nil {
		return err
	}
	if manager.GithubUsername == nil {
		return errors.New("github-username property not found.")
	}

	gitHubOrg := f.fetchGitHubOrg(todoParams, manager)
	if gitHubOrg == nil {
		return errors.New("function parameter is required when github-organization property is not set.")
	}

	// 1. root message 전송
	rootMessage := f.buildRootMessage(manager)
	messageID, err := f.channelSvc.WriteMessage(ctx, model.Group{
		ChannelID: fnCtx.Channel.ID,
		ID:        fnParams.Chat.ID,
	}, rootMessage)
	if err != nil {
		return err
	}

	// 2. thread message 전송
	assignedMessage, err := f.BuildAssignedPRMessage(ctx, fnCtx, *gitHubOrg, manager)
	if err != nil {
		return err
	}
	err = f.channelSvc.WriteThreadMessage(ctx, model.Group{
		ChannelID: fnCtx.Channel.ID,
		ID:        fnParams.Chat.ID,
	}, messageID, assignedMessage, false)
	if err != nil {
		return err
	}

	requestedReviewsMessage, err := f.BuildReviewRequestedPRMessage(ctx, fnCtx, *gitHubOrg, manager)
	if err != nil {
		return err
	}

	err = f.channelSvc.WriteThreadMessage(ctx, model.Group{
		ChannelID: fnCtx.Channel.ID,
		ID:        fnParams.Chat.ID,
	}, messageID, requestedReviewsMessage, false)
	if err != nil {
		return err
	}

	return nil
}

type TODOParams struct {
	GitHubOrganization string `json:"gitHubOrganization"`
}

func (f *TODOFunction) buildRootMessage(manager model.Manager) *model.Message {
	content := fmt.Sprintf(":four_leaf_clover: Hello %s. Here's your TODO\r\n", model.Mention(model.MentionTypeManager, manager.ID, manager.Name))
	return model.NewMessage(model.NewTextBlock(content))
}

func (f *TODOFunction) BuildReviewRequestedPRMessage(ctx context.Context, fnCtx appstore.Context, gitHubOrg string, manager model.Manager) (*model.Message, error) {
	installationID, err := f.githubSvc.FindAppInstallationID(ctx, gitHubOrg)
	if err != nil {
		return nil, err
	}
	if installationID == nil {
		return nil, nil
	}
	installationContext := github.NewInstallationContext(*installationID, gitHubOrg)

	issues, err := f.githubSvc.ListReviewRequestedPullRequest(ctx, installationContext, *manager.GithubUsername)
	if err != nil {
		return nil, err
	}

	var mdContent bytes.Buffer
	if len(issues) > 0 {
		mdContent.WriteString("## Here's pull requests waiting for your review.\r\n")
		for _, issue := range issues {
			if !issue.GetDraft() {
				// NOTE : search 결과에는 repository 정보가 함께 오지 않습니다.
				// 따라서, Repository URL 로부터 repo name 을 암묵적으로 파싱합니다.
				split := strings.Split(issue.GetRepositoryURL(), "/")
				repoName := split[len(split)-1]
				mdContent.WriteString(fmt.Sprintf("* [%s] [%s]( %s)\r\n", repoName, issue.GetTitle(), issue.GetHTMLURL()))
			}
		}
	} else {
		mdContent.WriteString("No Pull Requests review requested to you yet.")
	}

	messageBlocks, err := f.channelSvc.BuildMessageBlocksFromMarkdown(ctx, fnCtx.Channel.ID, mdContent.Bytes())
	if err != nil {
		return nil, err
	}

	return model.NewMessage(messageBlocks...), nil
}

func (f *TODOFunction) BuildAssignedPRMessage(ctx context.Context, fnCtx appstore.Context, gitHubOrg string, manager model.Manager) (*model.Message, error) {
	installationID, err := f.githubSvc.FindAppInstallationID(ctx, gitHubOrg)
	if err != nil {
		return nil, err
	}
	if installationID == nil {
		return nil, nil
	}
	installationContext := github.NewInstallationContext(*installationID, gitHubOrg)

	issues, err := f.githubSvc.ListAssignedPullRequest(ctx, installationContext, *manager.GithubUsername)
	if err != nil {
		return nil, err
	}

	var mdContent bytes.Buffer
	if len(issues) > 0 {
		mdContent.WriteString("## Here's your pull requests.\r\n")
		for _, issue := range issues {
			if !issue.GetDraft() && issue.GetState() == "open" {
				// NOTE : search 결과에는 repository 정보가 함께 오지 않습니다.
				// 따라서, Repository URL 로부터 repo name 을 암묵적으로 파싱합니다.
				split := strings.Split(issue.GetRepositoryURL(), "/")
				repoName := split[len(split)-1]
				mdContent.WriteString(fmt.Sprintf("* [%s] [%s]( %s)\r\n", repoName, issue.GetTitle(), issue.GetHTMLURL()))
			}
		}
	} else {
		mdContent.WriteString("No Pull Requests assigned to you yet.")
	}

	messageBlocks, err := f.channelSvc.BuildMessageBlocksFromMarkdown(ctx, fnCtx.Channel.ID, mdContent.Bytes())
	if err != nil {
		return nil, err
	}

	return model.NewMessage(messageBlocks...), nil
}

func (f *TODOFunction) fetchGitHubOrg(todoParam TODOParams, manager model.Manager) *string {
	if todoParam.GitHubOrganization != "" {
		return &todoParam.GitHubOrganization
	}
	return manager.GithubOrganization
}
