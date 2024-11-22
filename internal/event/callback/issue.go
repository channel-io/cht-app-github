package callback

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cbrgm/githubevents/githubevents"
	libgithub "github.com/google/go-github/v60/github"

	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/event/svc"
	"github.com/channel-io/cht-app-github/internal/github"
)

const (
	issueCommentCreatedTitleFormat = ":speech_balloon: %s %s commented by %s"
	issueOpenedTitle               = ":rotating_light: %s New issue opened! %s by %s"
	issueAssignedTitleFormat       = ":pray: %s assigned to %s"
	issueClosedTitleFormat         = ":x: %s closed by %s"
)

func NewIssueCommentCreated(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *IssueCommentCreated {
	return &IssueCommentCreated{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type IssueCommentCreated struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *IssueCommentCreated) Register(handler *githubevents.EventHandler) {
	handler.OnIssueCommentCreated(func(deliveryID string, eventName string, event *libgithub.IssueCommentEvent) error {
		if isSentFromBot(event.Sender) {
			return nil
		}

		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Organization.GetLogin(),
		)
		ctx := context.TODO()
		issueNumber := event.Issue.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *IssueCommentCreated) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.IssueCommentEvent) (*model.Message, error) {
	var mentionTexts bytes.Buffer
	if len(event.Issue.Assignees) > 0 {
		for i, assignee := range event.Issue.Assignees {
			if i > 0 {
				mentionTexts.WriteString(" ")
			}
			mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), assignee.GetLogin())
			if err != nil {
				return nil, err
			}
			mentionTexts.WriteString(mentionText)
		}
	} else {
		mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Issue.User.GetLogin())
		if err != nil {
			return nil, err
		}
		mentionTexts.WriteString(mentionText)
	}

	sender, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	var title string
	if event.Issue.IsPullRequest() {
		title = fmt.Sprintf(pullRequestReviewCommentedTitleFormat, mentionTexts.String(), model.InlineLink(event.Issue.GetHTMLURL(), "pull request"), sender)
	} else {
		title = fmt.Sprintf(issueCommentCreatedTitleFormat, mentionTexts.String(), model.InlineLink(event.Issue.GetHTMLURL(), "issue"), sender)
	}

	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewIssuesEventOpened(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *IssuesEventOpened {
	return &IssuesEventOpened{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type IssuesEventOpened struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *IssuesEventOpened) Register(handler *githubevents.EventHandler) {
	handler.OnIssuesEventOpened(func(deliveryID string, eventName string, event *libgithub.IssuesEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromIssue(event)
		issueNumber := event.Issue.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}

		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message, svc.WithoutTryFindingRootMessage())
	})
}

func (cb *IssuesEventOpened) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.IssuesEvent) (*model.Message, error) {
	mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(issueOpenedTitle, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), model.InlineLink(event.Issue.GetHTMLURL(), event.Issue.GetTitle()), mentionText)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewIssuesEventAssigned(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *IssuesEventAssigned {
	return &IssuesEventAssigned{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type IssuesEventAssigned struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *IssuesEventAssigned) Register(handler *githubevents.EventHandler) {
	handler.OnIssuesEventAssigned(func(deliveryID string, eventName string, event *libgithub.IssuesEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromIssue(event)
		issueNumber := event.Issue.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *IssuesEventAssigned) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.IssuesEvent) (*model.Message, error) {
	var mentionTexts bytes.Buffer
	if len(event.Issue.Assignees) > 0 {
		for i, assignee := range event.Issue.Assignees {
			if i > 0 {
				mentionTexts.WriteString(" ")
			}
			mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), assignee.GetLogin())
			if err != nil {
				return nil, err
			}
			mentionTexts.WriteString(mentionText)
		}
	} else {
		mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Issue.User.GetLogin())
		if err != nil {
			return nil, err
		}
		mentionTexts.WriteString(mentionText)
	}

	title := fmt.Sprintf(issueAssignedTitleFormat, model.InlineLink(event.Issue.GetHTMLURL(), "issue"), mentionTexts.String())
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewIssuesEventClosed(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *IssuesEventClosed {
	return &IssuesEventClosed{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type IssuesEventClosed struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *IssuesEventClosed) Register(handler *githubevents.EventHandler) {
	handler.OnIssuesEventClosed(func(deliveryID string, eventName string, event *libgithub.IssuesEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromIssue(event)
		issueNumber := event.Issue.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message, svc.WithBroadCasting())
	})
}

func (cb *IssuesEventClosed) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.IssuesEvent) (*model.Message, error) {
	mentionManager, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(issueClosedTitleFormat, model.InlineLink(event.Issue.GetHTMLURL(), "issue"), mentionManager)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func newGithubContextFromIssue(issue *libgithub.IssuesEvent) github.InstallationContext {
	return github.NewInstallationContext(
		issue.Installation.GetID(),
		issue.Org.GetLogin(),
	)
}
