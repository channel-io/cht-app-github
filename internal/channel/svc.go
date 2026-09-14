package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/channel-io/cht-app-github/internal/channel/client"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/channel/model/messageconv"
	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/pkg/cache"
)

type Service interface {
	FindManagerByGitHubMentionUsername(ctx context.Context, channelID string, username string) (*model.Manager, error)
	BuildMessageBlocksFromMarkdown(ctx context.Context, channelID string, markdown []byte) ([]model.MessageBlock, error)
	BuildTeamChatURL(group model.Group, rootMessageID string) string
	WriteMessage(ctx context.Context, group model.Group, message *model.Message) (messageID string, err error)
	WriteThreadMessage(
		ctx context.Context,
		group model.Group,
		rootMessageID string,
		message *model.Message,
		broadcast bool,
	) error
	FetchManagerByManagerID(ctx context.Context, channelID, managerID string) (model.Manager, error)
}

type ServiceImpl struct {
	client              client.Client
	githubUserNameCache ManagerCache
	managerIDCache      cache.Cache[model.Manager]

	deskURL string
}

func NewServiceImpl(client client.Client, conf *config.Config) *ServiceImpl {
	return &ServiceImpl{
		client:              client,
		githubUserNameCache: cache.NewLocalCache[map[string]model.Manager](),
		managerIDCache:      cache.NewLocalCache[model.Manager](),
		deskURL:             conf.ChannelTalk.DeskUrl,
	}
}

func (s *ServiceImpl) FindManagerByGitHubMentionUsername(ctx context.Context, channelID, username string) (*model.Manager, error) {
	key := strings.ToLower(username)

	managerMap, err := s.buildChannelManagersMap(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if manager, ok := managerMap[key]; ok {
		return &manager, nil
	}

	// Cache miss에는 Cache를 다시 만들어서 시도
	if managerMap, err = s.fetchChannelManagersMap(ctx, channelID); err != nil {
		return nil, err
	}
	if manager, ok := managerMap[key]; ok {
		return &manager, nil
	}
	return nil, nil
}

func (s *ServiceImpl) BuildMessageBlocksFromMarkdown(ctx context.Context, channelID string, markdown []byte) ([]model.MessageBlock, error) {
	managerMap, err := s.buildChannelManagersMap(ctx, channelID)
	if err != nil {
		return nil, err
	}
	return messageconv.FromGithubMarkdown(markdown, managerMap).Convert(), nil
}

func (s *ServiceImpl) buildChannelManagersMap(ctx context.Context, channelID string) (map[string]model.Manager, error) {
	cached, err := s.githubUserNameCache.Get(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return *cached, nil
	}
	return s.fetchChannelManagersMap(ctx, channelID)
}

// fetchChannelManagersMap 은 캐시를 무시하고 매니저 맵을 다시 만들어 캐시에 채운다.
func (s *ServiceImpl) fetchChannelManagersMap(ctx context.Context, channelID string) (map[string]model.Manager, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	managers, err := s.client.ListManagers(ctx, channelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to search managers")
	}

	m := make(map[string]model.Manager)
	for _, manager := range managers {
		if manager.GithubUsername != nil {
			m[strings.ToLower(*manager.GithubUsername)] = manager
			_ = s.managerIDCache.Set(ctx, manager.ID, manager, 60*time.Minute)
		} else {
			if lp := manager.GetEmailLocalPart(); lp != nil {
				m[*lp] = manager
			}
		}
	}

	if err = s.githubUserNameCache.Set(ctx, channelID, m, 10*time.Minute); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *ServiceImpl) WriteMessage(ctx context.Context, group model.Group, message *model.Message) (string, error) {
	return s.client.WriteGroupMessage(ctx, group.ChannelID, group.ID, message)
}

func (s *ServiceImpl) WriteThreadMessage(
	ctx context.Context,
	group model.Group,
	rootMessageID string,
	message *model.Message,
	broadcast bool,
) error {
	return s.client.WriteThreadMessage(ctx, group.ChannelID, group.ID, rootMessageID, message, broadcast)
}

func (s *ServiceImpl) BuildTeamChatURL(group model.Group, rootMessageID string) string {
	return fmt.Sprintf("%s/#/channels/%s/team_chats/groups/%s/%s", s.deskURL, group.ChannelID, group.ID, rootMessageID)
}

func (s *ServiceImpl) FetchManagerByManagerID(ctx context.Context, channelID, managerID string) (model.Manager, error) {
	cached, err := s.managerIDCache.Get(ctx, managerID)
	if err != nil {
		return model.Manager{}, err
	}

	if cached != nil {
		return *cached, nil
	}
	manager, err := s.client.GetManager(ctx, channelID, managerID)
	if err != nil {
		return model.Manager{}, err
	}
	_ = s.managerIDCache.Set(ctx, managerID, manager, 60*time.Minute)
	return manager, nil
}
