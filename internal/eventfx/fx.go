package eventfx

import (
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/internal/channel"
	"github.com/channel-io/cht-app-github/internal/event"
	"github.com/channel-io/cht-app-github/internal/event/callback"
	"github.com/channel-io/cht-app-github/internal/event/svc"
)

var Option = fx.Options(
	fx.Provide(
		fx.Annotate(
			channel.NewServiceImpl,
			fx.As(new(channel.Service)),
		),
		fx.Annotate(
			event.NewGithubEventHandler,
			fx.ParamTags("", "", `group:"event.callbacks"`),
		),
	),

	fx.Provide(
		// Issue
		eventCallback(callback.NewIssueCommentCreated),
		eventCallback(callback.NewIssuesEventOpened),
		eventCallback(callback.NewIssuesEventAssigned),
		eventCallback(callback.NewIssuesEventClosed),

		// Pull Request
		eventCallback(callback.NewPullRequestEventReadyForReview),
		eventCallback(callback.NewPullRequestEventOpened),
		eventCallback(callback.NewPullRequestEventClosed),
		eventCallback(callback.NewPullRequestReviewEventSubmitted),
		eventCallback(callback.NewPullRequestEventReviewRequested),
		eventCallback(callback.NewPullRequestEventReviewRequestRemoved),
		eventCallback(callback.NewPullRequestEventAssigned),
		eventCallback(callback.NewPullRequestEventSynchronize),

		eventCallback(callback.NewReleaseEventReleased),
		eventCallback(callback.NewStatusEventAny),
	),

	fx.Provide(
		svc.NewIssueSvc,
		svc.NewCommonSvc,
		svc.NewStatusSvc,
		svc.NewReleaseSvc,
	),
)

func eventCallback(fn interface{}) interface{} {
	return fx.Annotate(
		fn,
		fx.As(new(event.EventCallback)),
		fx.ResultTags(`group:"event.callbacks"`),
	)
}
