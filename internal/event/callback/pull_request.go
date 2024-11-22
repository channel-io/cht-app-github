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
	pullRequestBodyFormat                 = "%s (%s → %s)"
	pullRequestOpenTitleFormat            = ":writing_hand: %s New pull request opened! by %s"
	pullRequestReadyForReviewTitleFormat  = ":fire: %s %s ready for review!"
	pullRequestDraftTitleFormat           = ":building_construction: %s Draft pull request created by %s"
	pullRequestMergedTitleFormat          = ":white_check_mark: %s pull request merged! by %s"
	pullRequestClosedTitleFormat          = ":boom: %s pull request closed... by %s"
	pullRequestAssignedTitleFormat        = ":pray: %s assigned to %s"
	pullRequestSynchronizedTitleFormat    = ":arrows_counterclockwise: %s has been updated by %s"
	pullRequestReviewCommentedTitleFormat = ":thinking_face::speech_balloon: %s %s commented by %s"
	pullRequestReviewApprovedTitleFormat  = ":100: %s %s approved! by %s"
	pullRequestReviewRequestedTitleFormat = ":pray: %s %s review requested by %s"
	pullRequestReviewRequestRemovedFormat = ":x: %s %s review request removed by %s"
)

func NewPullRequestEventReadyForReview(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventReadyForReview {
	return &PullRequestEventReadyForReview{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventReadyForReview struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventReadyForReview) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventReadyForReview(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *PullRequestEventReadyForReview) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	var mentionTexts bytes.Buffer
	for i, reviewer := range event.PullRequest.RequestedReviewers {
		if i > 0 {
			mentionTexts.WriteString(" ")
		}
		mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), reviewer.GetLogin())
		if err != nil {
			return nil, err
		}
		mentionTexts.WriteString(mentionText)
	}
	title := fmt.Sprintf(pullRequestReadyForReviewTitleFormat, mentionTexts.String(), model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"))
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewPullRequestEventOpened(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventOpened {
	return &PullRequestEventOpened{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventOpened struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventOpened) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventOpened(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)
		issueNumber := event.PullRequest.GetNumber()

		if !isSentFromBot(event.GetSender()) && len(event.PullRequest.Assignees) == 0 {
			assignees := []string{event.GetSender().GetLogin()}
			if err := cb.issueSvc.AddAssigneeToIssue(ctx, installCtx, event.Repo.GetName(), issueNumber, assignees); err != nil {
				return err
			}
		}

		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}

		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message, svc.WithoutTryFindingRootMessage())
	})
}

func (cb *PullRequestEventOpened) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	var title string
	mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}
	if event.PullRequest.GetDraft() {
		title = fmt.Sprintf(pullRequestDraftTitleFormat, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), mentionText)
	} else {
		title = fmt.Sprintf(pullRequestOpenTitleFormat, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), mentionText)
	}

	return model.NewMessage(
		model.NewTextBlock(title),
		model.NewTextBlock(bodyContentForPullRequest(event.PullRequest)),
	), nil
}

func NewPullRequestEventClosed(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventClosed {
	return &PullRequestEventClosed{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventClosed struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventClosed) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventClosed(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message, svc.WithBroadCasting())
	})
}

func (cb *PullRequestEventClosed) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}
	var title string
	if event.PullRequest.GetMerged() {
		title = fmt.Sprintf(pullRequestMergedTitleFormat, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), mentionText)
	} else {
		title = fmt.Sprintf(pullRequestClosedTitleFormat, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), mentionText)
	}

	return model.NewMessage(

		model.NewTextBlock(title),
		model.NewTextBlock(bodyContentForPullRequest(event.PullRequest)),
	), nil
}

func NewPullRequestReviewEventSubmitted(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestReviewEventSubmitted {
	return &PullRequestReviewEventSubmitted{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestReviewEventSubmitted struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestReviewEventSubmitted) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestReviewEventSubmitted(func(deliveryID string, eventName string, event *libgithub.PullRequestReviewEvent) error {
		if event.PullRequest.GetDraft() {
			return nil
		}

		if isSentFromBot(event.Sender) {
			return nil
		}
		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Organization.GetLogin(),
		)
		ctx := context.TODO()
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *PullRequestReviewEventSubmitted) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestReviewEvent) (*model.Message, error) {
	var mentionTexts bytes.Buffer
	if len(event.PullRequest.Assignees) > 0 {
		for i, assignee := range event.PullRequest.Assignees {
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
		mentionText, err := cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.PullRequest.User.GetLogin())
		if err != nil {
			return nil, err
		}
		mentionTexts.WriteString(mentionText)
	}

	senderManager, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}
	var title string
	if event.Review.GetState() == "approved" {
		title = fmt.Sprintf(pullRequestReviewApprovedTitleFormat, mentionTexts.String(), model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), senderManager)
	} else {
		title = fmt.Sprintf(pullRequestReviewCommentedTitleFormat, mentionTexts.String(), model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), senderManager)
	}

	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewPullRequestEventReviewRequested(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventReviewRequested {
	return &PullRequestEventReviewRequested{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventReviewRequested struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventReviewRequested) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventReviewRequested(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		// CODE OWNER 로 team 이 설정된 경우, event.RequestedTeam 으로 요청이 옴. 이런 경우, 멘션을 할 수 없기에 우선 ignore 해봄.
		// 추후 팀멘션 기능 추가시 고려 대상임.
		if event.RequestedReviewer == nil {
			return nil
		}

		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)

		if cb.commonSvc.IgnoreBot(ctx, event.Organization.GetLogin(), event.Repo.GetName()) && event.Sender.GetType() == "Bot" {
			return nil
		}

		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *PullRequestEventReviewRequested) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	var reviewer string
	var err error
	if event.PullRequest.GetDraft() {
		reviewer, err = cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.RequestedReviewer.GetLogin())
		if err != nil {
			return nil, err
		}
	} else {
		reviewer, err = cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.RequestedReviewer.GetLogin())
		if err != nil {
			return nil, err
		}
	}
	requester, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(pullRequestReviewRequestedTitleFormat, reviewer, model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), requester)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewPullRequestEventReviewRequestRemoved(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventReviewRequestRemoved {
	return &PullRequestEventReviewRequestRemoved{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventReviewRequestRemoved struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventReviewRequestRemoved) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventReviewRequestRemoved(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		// 팀이 리뷰 요청에서 삭제되는 경우, requested_reviewer 는 비어있고 requested_team 에 값이 들어옴.
		// 추후 팀멘션 기능 추가시 고려 대상임.
		if event.RequestedReviewer == nil {
			return nil
		}

		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *PullRequestEventReviewRequestRemoved) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	var reviewer string
	var err error

	if event.PullRequest.GetDraft() {
		reviewer, err = cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.RequestedReviewer.GetLogin())
		if err != nil {
			return nil, err
		}
	} else {
		reviewer, err = cb.commonSvc.BuildManagerMentionTextByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.RequestedReviewer.GetLogin())
		if err != nil {
			return nil, err
		}
	}
	requester, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(pullRequestReviewRequestRemovedFormat, reviewer, model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), requester)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewPullRequestEventAssigned(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventAssigned {
	return &PullRequestEventAssigned{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventAssigned struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventAssigned) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventAssigned(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Organization.GetLogin(),
		)
		ctx := context.TODO()
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message, svc.StopWithoutRootMessage())
	})
}

func (cb *PullRequestEventAssigned) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {

	assignee, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Assignee.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(pullRequestAssignedTitleFormat, model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), assignee)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func NewPullRequestEventSynchronize(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc) *PullRequestEventSynchronize {
	return &PullRequestEventSynchronize{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
	}
}

type PullRequestEventSynchronize struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
}

func (cb *PullRequestEventSynchronize) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestEventSynchronize(func(deliveryID string, eventName string, event *libgithub.PullRequestEvent) error {
		ctx := context.TODO()
		installCtx := newGithubContextFromPullRequest(event)
		issueNumber := event.PullRequest.GetNumber()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, event.Repo.GetName(), issueNumber, message)
	})
}

func (cb *PullRequestEventSynchronize) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestEvent) (*model.Message, error) {
	sender, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(pullRequestSynchronizedTitleFormat, model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"), sender)
	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

func bodyContentForPullRequest(pr *libgithub.PullRequest) string {
	return fmt.Sprintf(pullRequestBodyFormat, model.InlineLink(pr.GetHTMLURL(), pr.GetTitle()), pr.Head.GetLabel(), pr.Base.GetLabel())
}

func newGithubContextFromPullRequest(pr *libgithub.PullRequestEvent) github.InstallationContext {
	return github.NewInstallationContext(
		pr.Installation.GetID(),
		pr.Organization.GetLogin(),
	)
}
