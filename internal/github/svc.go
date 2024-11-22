package github

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/samber/lo"

	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/config"
	"github.com/channel-io/cht-app-github/pkg/cache"
	"github.com/channel-io/cht-app-github/pkg/utils"
)

const (
	rootMessageIdCacheKey = "RootMessageID"
)

type Service interface {
	// channel talk integration
	FindRootMessageID(ctx context.Context, installCtx InstallationContext, repository string, issueNumber int, retry int) (*string, error)
	FindGroup(ctx context.Context, ghContext InstallationContext, repository string) (model.Group, error)
	FindReleaseGroup(ctx context.Context, ghContext InstallationContext, repository string) (model.Group, error)

	CreateComment(ctx context.Context, installCtx InstallationContext, repository string, number int, body string) error
	ListPullRequestNumberByCommitSHA(ctx context.Context, installCtx InstallationContext, repoName, sha string, predicates ...FilterPullRequestPredicate) ([]*github.PullRequest, error)
	FetchPullRequest(ctx context.Context, installCtx InstallationContext, repository string, number int) (*github.PullRequest, error)
	AddAssigneeToIssue(ctx context.Context, installCtx InstallationContext, repository string, number int, assignees []string) error
	FindAppInstallationID(ctx context.Context, org string) (*int64, error)
	ListReviewRequestedPullRequest(ctx context.Context, installCtx InstallationContext, user string) ([]*github.Issue, error)
	ListAssignedPullRequest(ctx context.Context, installCtx InstallationContext, user string) ([]*github.Issue, error)
}

type ServiceImpl struct {
	githubAppID       int64
	channelIDKey      string
	groupIDKey        string
	releaseGroupIDKey string
	privateKey        []byte

	installationClientPool map[InstallationContext]*InstallationClient
	appClient              *AppClient
	customPropertyCache    Cache[string]
	installationIDCache    Cache[int64]
	metrics                *ClientMetrics
}

func NewServiceImpl(conf *config.Config, metrics *ClientMetrics) *ServiceImpl {
	privateKey, err := os.ReadFile(conf.Github.App.PrivateKeyPath)
	if err != nil {
		// FIXME: this leads to runtime error when priv key is not provided
		// log.Fatalf("private key is required: %s", conf.Github.App.PrivateKeyPath)
		privateKey = nil
	}

	// NOTE : private key 가 없는 경우에 대해서, early fail 처리를 위에서 하고 있지 않음. 이를 그대로 유지함.
	//appClient, err := newAppClient(conf.Github.App.Id, privateKey, metrics)
	//if err != nil {
	//	log.Fatalf("failed to generate app client")
	//}

	return &ServiceImpl{
		githubAppID:            conf.Github.App.Id,
		channelIDKey:           conf.Github.Properties.ChannelIdKey,
		groupIDKey:             conf.Github.Properties.GroupIdKey,
		releaseGroupIDKey:      conf.Github.Properties.ReleaseGroupIdKey,
		privateKey:             privateKey,
		installationClientPool: make(map[InstallationContext]*InstallationClient),
		//appClient:              appClient,
		customPropertyCache: cache.NewLocalCache[string](),
		installationIDCache: cache.NewLocalCache[int64](),
		metrics:             metrics,
	}
}

func (s *ServiceImpl) FindRootMessageID(ctx context.Context, installCtx InstallationContext, repository string, issueNumber int, tryCount int) (*string, error) {
	for {
		if tryCount <= 0 {
			break
		}
		messageId, err := s.findMessageIdFromComments(ctx, installCtx, repository, issueNumber)
		if err != nil {
			return nil, err
		}
		if messageId != nil {
			return messageId, nil
		}

		tryCount--
		time.Sleep(1 * time.Second)
	}
	return nil, nil
}

func (s *ServiceImpl) findMessageIdFromComments(ctx context.Context, installCtx InstallationContext, repository string, number int) (*string, error) {
	cached, err := s.customPropertyCache.Get(ctx, s.cacheKeyForIssue(installCtx, repository, number, rootMessageIdCacheKey))
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	textBody, err := s.findCommentTextWrittenByApp(ctx, installCtx, repository, number)
	if err != nil {
		return nil, err
	}
	if textBody != nil {
		parsed := utils.ParseMessageIdFromTeamChatDeskUrl(config.Get().ChannelTalk.DeskUrl, *textBody)
		if parsed != "" {
			_ = s.customPropertyCache.Set(ctx, s.cacheKeyForIssue(installCtx, repository, number, rootMessageIdCacheKey), parsed, 120*time.Minute)
			return &parsed, nil
		}
	}

	return nil, nil
}

func (s *ServiceImpl) FindGroup(ctx context.Context, installCtx InstallationContext, repository string) (model.Group, error) {
	channelID, err := s.findCustomProperty(ctx, installCtx, repository, s.channelIDKey)
	if err != nil {
		return model.Group{}, err
	}

	groupID, err := s.findCustomProperty(ctx, installCtx, repository, s.groupIDKey)
	if err != nil {
		return model.Group{}, err
	}

	return model.Group{
		ChannelID: channelID,
		ID:        groupID,
	}, nil
}

func (s *ServiceImpl) FindReleaseGroup(ctx context.Context, installCtx InstallationContext, repository string) (model.Group, error) {
	channelID, err := s.findCustomProperty(ctx, installCtx, repository, s.channelIDKey)
	if err != nil {
		return model.Group{}, err
	}

	groupID, err := s.findCustomProperty(ctx, installCtx, repository, s.releaseGroupIDKey)
	if err != nil {
		return model.Group{}, err
	}

	return model.Group{
		ChannelID: channelID,
		ID:        groupID,
	}, nil
}

func (s *ServiceImpl) findCustomProperty(ctx context.Context, installCtx InstallationContext, repository, key string) (string, error) {
	cached, err := s.customPropertyCache.Get(ctx, s.cacheKeyForRepository(installCtx, repository, key))
	if err != nil {
		return "", err
	}
	if cached != nil {
		return *cached, nil
	}

	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return "", err
	}
	value, err := client.FindCustomProperty(ctx, repository, key)
	if err != nil {
		return "", err
	}
	_ = s.customPropertyCache.Set(ctx, s.cacheKeyForRepository(installCtx, repository, key), value, 60*time.Minute)
	return value, nil
}

func (s *ServiceImpl) CreateComment(ctx context.Context, installCtx InstallationContext, repository string, number int, body string) error {
	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return err
	}
	return client.CreateCommentOnIssue(ctx, installCtx.OrgLogin, repository, number, body)
}

func (s *ServiceImpl) ListPullRequestNumberByCommitSHA(ctx context.Context, installCtx InstallationContext, repoName, sha string, predicates ...FilterPullRequestPredicate) ([]*github.PullRequest, error) {
	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return nil, err
	}
	pullRequests, _, err := client.PullRequests.ListPullRequestsWithCommit(ctx, installCtx.OrgLogin, repoName, sha, nil)
	if err != nil {
		return nil, err
	}

	if len(predicates) > 0 {
		pullRequests = lo.Filter(pullRequests, func(item *github.PullRequest, _ int) bool {
			for _, pred := range predicates {
				if !pred(item) {
					return false
				}
			}
			return true
		})
	}

	if len(pullRequests) == 0 {
		return nil, nil
	}

	return pullRequests, nil
}

func (s *ServiceImpl) FetchPullRequest(ctx context.Context, installCtx InstallationContext, repository string, number int) (*github.PullRequest, error) {
	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return nil, err
	}
	issue, _, err := client.PullRequests.Get(ctx, installCtx.OrgLogin, repository, number)
	if err != nil {
		return nil, err
	}
	return issue, nil
}

func (s *ServiceImpl) AddAssigneeToIssue(ctx context.Context, installCtx InstallationContext, repository string, number int, assignees []string) error {
	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return err
	}
	return client.AddAssigneeToIssue(ctx, repository, number, assignees)
}

func (s *ServiceImpl) getInstallationClient(installCtx InstallationContext) (*InstallationClient, error) {
	client, exists := s.installationClientPool[installCtx]
	if !exists {
		client, err := newGithubClientWithInstallation(s.githubAppID, s.privateKey, installCtx)
		if err != nil {
			return nil, err
		}
		s.installationClientPool[installCtx] = newInstallationClient(client, installCtx, s.metrics)
		return s.installationClientPool[installCtx], nil
	}
	return client, nil
}

func (s *ServiceImpl) findCommentTextWrittenByApp(ctx context.Context, installCtx InstallationContext, repository string, number int) (*string, error) {
	client, err := s.getInstallationClient(installCtx)
	if err != nil {
		return nil, err
	}
	comments, err := client.FindAllCommentsOnIssue(ctx, repository, number)
	if err != nil {
		return nil, err
	}
	for i := range comments {
		if utils.IsChannelTeamChatUriFormat(config.Get().ChannelTalk.DeskUrl, comments[i].GetBody()) {
			return comments[i].Body, nil
		}
	}
	return nil, nil
}

func (s *ServiceImpl) cacheKeyForIssue(installCtx InstallationContext, repository string, number int, key string) string {
	return fmt.Sprintf("%s:%s:%d:%s", installCtx.OrgLogin, repository, number, key)
}

func (s *ServiceImpl) cacheKeyForRepository(installCtx InstallationContext, repository, key string) string {
	return fmt.Sprintf("%s:%s:%s", installCtx.OrgLogin, repository, key)
}

func (s *ServiceImpl) cacheKeyForInstallationID(org string) string {
	return fmt.Sprintf("installationid:%s", org)
}

func (s *ServiceImpl) FindAppInstallationID(ctx context.Context, org string) (*int64, error) {
	cached, err := s.installationIDCache.Get(ctx, s.cacheKeyForInstallationID(org))
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return cached, nil
	}

	// NOTE : private key 가 없는 경우에 대해서, early fail 처리를 위에서 하고 있지 않고 있기에, 이곳에서 fail 함.
	if s.appClient == nil {
		appClient, err := newAppClient(s.githubAppID, s.privateKey, s.metrics)
		if err != nil {
			log.Fatalf("failed to generate app client")
		}
		s.appClient = appClient
	}

	installations, err := s.appClient.ListInstallations(ctx)
	if err != nil {
		return nil, err
	}
	for i := range installations {
		if installations[i].Account.GetType() == "Organization" {
			err := s.installationIDCache.Set(ctx, s.cacheKeyForInstallationID(org), installations[i].GetID(), -1)
			if err != nil {
				return nil, err
			}
		}
	}
	return s.installationIDCache.Get(ctx, s.cacheKeyForInstallationID(org))
}

func (s *ServiceImpl) ListReviewRequestedPullRequest(ctx context.Context, installCtx InstallationContext, user string) ([]*github.Issue, error) {
	installationClient, err := s.getInstallationClient(installCtx)
	if err != nil {
		return nil, err
	}

	return installationClient.ListReviewRequestedPullRequests(ctx, user)
}

func (s *ServiceImpl) ListAssignedPullRequest(ctx context.Context, installCtx InstallationContext, user string) ([]*github.Issue, error) {
	installationClient, err := s.getInstallationClient(installCtx)
	if err != nil {
		return nil, err
	}

	return installationClient.ListOpenedPullRequests(ctx, user)
}
