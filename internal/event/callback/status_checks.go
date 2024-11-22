package callback

import (
	"context"
	"fmt"

	"github.com/cbrgm/githubevents/githubevents"
	libgithub "github.com/google/go-github/v60/github"

	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/event/svc"
	"github.com/channel-io/cht-app-github/internal/github"
)

const (
	statusResultFormat = ":arrows_counterclockwise: %s pipeline with %s has been %s"
)

func NewStatusEventAny(commonSvc *svc.CommonSvc, statusSvc *svc.StatusSvc) *StatusChecksEventAny {
	return &StatusChecksEventAny{
		commonSvc: commonSvc,
		statusSvc: statusSvc,
	}
}

type StatusChecksEventAny struct {
	commonSvc *svc.CommonSvc
	statusSvc *svc.StatusSvc
}

func (cb *StatusChecksEventAny) Register(handler *githubevents.EventHandler) {
	handler.OnStatusEventAny(func(deliveryID string, eventName string, event *libgithub.StatusEvent) error {
		ctx := context.TODO()
		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Org.GetLogin())
		message, err := cb.buildStatusMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		if message == nil {
			return nil
		}
		return cb.statusSvc.SyncCommitStatusWithChannelTalk(ctx, installCtx, event.Repo.GetName(), event.GetSHA(), message)
	})

	handler.OnCheckRunEventCompleted(func(deliveryID string, eventName string, event *libgithub.CheckRunEvent) error {
		// NOTE: check_suite 의 상태가 completed 인 경우에만 check_run 의 결과를 메시지로 보냅니다.
		// ref) https://docs.github.com/en/rest/guides/using-the-rest-api-to-interact-with-checks?apiVersion=2022-11-28#about-check-suites
		if event.GetCheckRun().GetCheckSuite().GetStatus() != "completed" {
			return nil
		}
		ctx := context.TODO()
		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Org.GetLogin())
		message, err := cb.buildCheckRunMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		if message == nil {
			return nil
		}
		return cb.statusSvc.SyncCommitStatusWithChannelTalk(ctx, installCtx, event.Repo.GetName(), event.CheckRun.GetHeadSHA(), message)
	})
}

func (cb *StatusChecksEventAny) buildStatusMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.StatusEvent) (*model.Message, error) {
	var title string
	commitLink := model.InlineLink(event.GetTargetURL(), event.Commit.GetSHA()[:10])
	switch event.GetState() {
	case "success", "error", "failure":
		title = fmt.Sprintf(statusResultFormat, event.GetContext(), commitLink, event.GetState())
	default:
		// https://docs.github.com/en/webhooks/webhook-events-and-payloads#status
		// pending state 는 무시합니다.
		return nil, nil
	}

	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}

// TODO : event.CheckRun.PullRequests 에 PullRequest 가 들어가 있는 경우는??
func (cb *StatusChecksEventAny) buildCheckRunMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.CheckRunEvent) (*model.Message, error) {
	var title string
	switch event.CheckRun.GetConclusion() {
	case "success", "cancelled", "failure", "timed_out":
		title = fmt.Sprintf(statusResultFormat, event.CheckRun.GetName(), event.CheckRun.App.GetName(), event.CheckRun.GetConclusion())
	default:
		// 외 conclusion 은 다음을 참고합니다.
		// https://docs.github.com/en/webhooks/webhook-events-and-payloads#check_run
		return nil, nil
	}

	return model.NewMessage(
		model.NewTextBlock(title),
	), nil
}
