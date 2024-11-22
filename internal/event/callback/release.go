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
	releasedTitleFormat = ":package: %s: %s released by %s"
)

func NewReleaseEventReleased(commonSvc *svc.CommonSvc, releaseSvc *svc.ReleaseSvc) *ReleaseEventReleased {
	return &ReleaseEventReleased{
		commonSvc:  commonSvc,
		releaseSvc: releaseSvc,
	}
}

type ReleaseEventReleased struct {
	commonSvc  *svc.CommonSvc
	releaseSvc *svc.ReleaseSvc
}

func (cb *ReleaseEventReleased) Register(handler *githubevents.EventHandler) {
	handler.OnReleaseEventReleased(func(deliveryID string, eventName string, event *libgithub.ReleaseEvent) error {
		installCtx := github.NewInstallationContext(
			event.Installation.GetID(),
			event.Org.GetLogin())

		ctx := context.TODO()
		message, err := cb.buildMessage(ctx, installCtx, event)
		if err != nil {
			return err
		}
		return cb.releaseSvc.SyncReleaseWithChannelTalk(ctx, installCtx, event.Repo.GetName(), message)
	})
}

func (cb *ReleaseEventReleased) buildMessage(ctx context.Context, installCtx github.InstallationContext, event *libgithub.ReleaseEvent) (*model.Message, error) {

	mentionManager, err := cb.commonSvc.FindManagerNameByGithubUsername(ctx, installCtx, event.Repo.GetName(), event.Sender.GetLogin())
	if err != nil {
		return nil, err
	}
	title := fmt.Sprintf(releasedTitleFormat, model.InlineLink(event.Repo.GetHTMLURL(), event.Repo.GetName()), model.InlineLink(event.Release.GetHTMLURL(), event.Release.GetTagName()), mentionManager)
	blocksFromBody, err := cb.releaseSvc.BuildMessageBlocksFromBody(ctx, installCtx, event.Repo.GetName(), event.Release.GetBody())
	if err != nil {
		return nil, err
	}

	blocks := []model.MessageBlock{
		model.NewTextBlock(title),
	}
	blocks = append(blocks, blocksFromBody...)
	return model.NewMessage(blocks...), nil
}
