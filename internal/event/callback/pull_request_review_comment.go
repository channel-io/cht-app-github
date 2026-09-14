package callback

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cbrgm/githubevents/githubevents"
	libgithub "github.com/google/go-github/v60/github"

	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/event/svc"
	"github.com/channel-io/cht-app-github/internal/github"
)

const (
	pullRequestReviewCommentInlineTitleFormat = ":thinking_face::speech_balloon: %s %s %s commented by %s"
	inlineCommentDedupTTL                     = 1 * time.Hour
)

func NewPullRequestReviewCommentEventCreated(commonSvc *svc.CommonSvc, issueSvc *svc.IssueSvc, githubSvc github.Service) *PullRequestReviewCommentEventCreated {
	return &PullRequestReviewCommentEventCreated{
		commonSvc: commonSvc,
		issueSvc:  issueSvc,
		githubSvc: githubSvc,
	}
}

type PullRequestReviewCommentEventCreated struct {
	commonSvc *svc.CommonSvc
	issueSvc  *svc.IssueSvc
	githubSvc github.Service
	// processedReviewIDs claims each review_id at most once so that a batched
	// review submission with N inline comments yields at most one inline
	// notification. Entries auto-expire after inlineCommentDedupTTL.
	// Single-replica assumption.
	processedReviewIDs sync.Map
}

func (cb *PullRequestReviewCommentEventCreated) Register(handler *githubevents.EventHandler) {
	handler.OnPullRequestReviewCommentEventCreated(func(deliveryID string, eventName string, event *libgithub.PullRequestReviewCommentEvent) error {
		if isSentFromBot(event.Sender) {
			return nil
		}

		reviewID := event.Comment.GetPullRequestReviewID()
		if reviewID != 0 {
			if _, claimed := cb.processedReviewIDs.LoadOrStore(reviewID, struct{}{}); claimed {
				return nil
			}
			cb.scheduleDedupCleanup(reviewID)
		}

		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Org.GetLogin(),
		)
		ctx := context.TODO()
		repo := event.Repo.GetName()
		prNumber := event.PullRequest.GetNumber()

		// Wrapper body 가 있으면 PullRequestReviewEventSubmitted 핸들러가 처리하므로
		// inline 알림은 보내지 않는다 (사용자 피드백 반영: body 없을 때만 inline 노출).
		if reviewID != 0 {
			review, err := cb.githubSvc.FetchReview(ctx, installCtx, repo, prNumber, reviewID)
			if err != nil {
				cb.processedReviewIDs.Delete(reviewID)
				return err
			}
			if review.GetBody() != "" {
				return nil
			}
		}

		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.issueSvc.SyncIssueWithChannelTalk(ctx, installCtx, repo, prNumber, message)
	})
}

func (cb *PullRequestReviewCommentEventCreated) scheduleDedupCleanup(reviewID int64) {
	time.AfterFunc(inlineCommentDedupTTL, func() {
		cb.processedReviewIDs.Delete(reviewID)
	})
}

func (cb *PullRequestReviewCommentEventCreated) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.PullRequestReviewCommentEvent) (*model.Message, error) {
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

	sender, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}

	fileRef := buildCommentFileRef(event.Comment)
	title := fmt.Sprintf(
		pullRequestReviewCommentInlineTitleFormat,
		mentionTexts.String(),
		model.InlineLink(event.PullRequest.GetHTMLURL(), "pull request"),
		model.InlineLink(event.Comment.GetHTMLURL(), fileRef),
		sender,
	)

	blocks := []model.MessageBlock{model.NewTextBlock(title)}
	if body := truncateRunes(event.Comment.GetBody(), commentBodyMaxRunes); body != "" {
		blocks = append(blocks, model.NewTextBlock(model.EscapedString(body)))
	}
	return model.NewMessage(blocks...), nil
}

func buildCommentFileRef(comment *libgithub.PullRequestComment) string {
	path := comment.GetPath()
	if path == "" {
		return "comment"
	}
	if line := comment.GetLine(); line > 0 {
		return fmt.Sprintf("%s:L%d", path, line)
	}
	return path
}
