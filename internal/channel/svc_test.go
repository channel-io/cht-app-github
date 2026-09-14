package channel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/channel-io/cht-app-github/internal/channel/client"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/config"
)

func TestServiceImpl_buildGithubManagersMap(t *testing.T) {
	tests := []struct {
		name string
		// managers is mock return value for client request
		managers []model.Manager
		expected map[string]model.Manager
	}{
		{
			name: "basic",
			managers: []model.Manager{
				{
					ID:             "1",
					Name:           "Claud",
					GithubUsername: ptrString("ch-claud"),
				},
				{
					ID:             "2",
					Name:           "Dylan",
					GithubUsername: ptrString("ch-dylan"),
				},
				{
					ID:             "3",
					Name:           "Lento",
					GithubUsername: ptrString("ch-lento"),
				},
			},
			expected: map[string]model.Manager{
				"ch-claud": {
					ID:             "1",
					Name:           "Claud",
					GithubUsername: ptrString("ch-claud"),
				},
				"ch-dylan": {
					ID:             "2",
					Name:           "Dylan",
					GithubUsername: ptrString("ch-dylan"),
				},
				"ch-lento": {
					ID:             "3",
					Name:           "Lento",
					GithubUsername: ptrString("ch-lento"),
				},
			},
		},
		{
			name: "fallback to email local part and github id map",
			managers: []model.Manager{
				{
					ID:             "1",
					Name:           "Claud",
					Email:          ptrString("claud-33@channel.io"),
					GithubUsername: nil,
				},
				{
					ID:             "2",
					Name:           "Dylan",
					GithubUsername: ptrString("ch-dylan"),
				},
				{
					ID:             "3",
					Name:           "Lento",
					GithubUsername: ptrString("ch-lento"),
				},
			},
			expected: map[string]model.Manager{
				"claud-33": {
					ID:             "1",
					Name:           "Claud",
					Email:          ptrString("claud-33@channel.io"),
					GithubUsername: nil,
				},
				"ch-dylan": {
					ID:             "2",
					Name:           "Dylan",
					GithubUsername: ptrString("ch-dylan"),
				},
				"ch-lento": {
					ID:             "3",
					Name:           "Lento",
					GithubUsername: ptrString("ch-lento"),
				},
			},
		},
		{
			name:     "no managers",
			managers: []model.Manager{},
			expected: map[string]model.Manager{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := new(mockClient)
			m.On("ListManagers", mock.Anything, mock.Anything).
				Return(tc.managers, nil)

			s := NewServiceImpl(m, new(config.Config))

			ctx := context.TODO()
			channelID := "1"
			actual, err := s.buildChannelManagersMap(ctx, channelID)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestServiceImpl_buildGithubManagersMap_Cache(t *testing.T) {
	t.Parallel()

	mockManagers := []model.Manager{
		{
			ID:             "1",
			Name:           "Claud",
			GithubUsername: ptrString("ch-claud"),
		},
		{
			ID:             "2",
			Name:           "Dylan",
			GithubUsername: ptrString("ch-dylan"),
		},
		{
			ID:             "3",
			Name:           "Lento",
			GithubUsername: ptrString("ch-lento"),
		},
	}

	m := new(mockClient)
	m.On("ListManagers", mock.Anything, mock.Anything).
		Return(mockManagers, nil)

	c := new(mockManagerCache)
	c.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	s := NewServiceImpl(m, new(config.Config))

	ctx := context.TODO()
	channelID := "1"
	expected := map[string]model.Manager{
		"ch-claud": {
			ID:             "1",
			Name:           "Claud",
			GithubUsername: ptrString("ch-claud"),
		},
		"ch-dylan": {
			ID:             "2",
			Name:           "Dylan",
			GithubUsername: ptrString("ch-dylan"),
		},
		"ch-lento": {
			ID:             "3",
			Name:           "Lento",
			GithubUsername: ptrString("ch-lento"),
		},
	}

	// first request
	getCall := c.On("Get", mock.Anything, channelID).Return((*map[string]model.Manager)(nil), nil) // cache miss

	actual, err := s.buildChannelManagersMap(ctx, channelID)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
	m.AssertCalled(t, "ListManagers", mock.Anything, channelID)
	m.AssertNumberOfCalls(t, "ListManagers", 1)

	// following request
	getCall.Unset()
	c.On("Get", mock.Anything, mock.Anything).Return(&expected, nil) // cache hit

	actual, err = s.buildChannelManagersMap(ctx, channelID)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
	m.AssertNumberOfCalls(t, "ListManagers", 1) // no extra client call
}

func TestServiceImpl_buildGithubManagersMap_ClientError(t *testing.T) {
	t.Parallel()

	m := new(mockClient)
	m.On("ListManagers", mock.Anything, mock.Anything).
		Return(([]model.Manager)(nil), assert.AnError)

	s := NewServiceImpl(m, new(config.Config))

	ctx := context.TODO()
	channelID := "1"
	res, err := s.buildChannelManagersMap(ctx, channelID)
	assert.Nil(t, res)
	assert.Error(t, err)
}

func TestServiceImpl_BuildTeamChatURL(t *testing.T) {
	tests := []struct {
		name          string
		svc           *ServiceImpl
		group         model.Group
		rootMessageID string
		expected      string
	}{
		{
			name: "basic",
			svc: &ServiceImpl{
				deskURL: "https://desk.exp.channel.io",
			},
			group: model.Group{
				ChannelID: "1",
				ID:        "23",
			},
			rootMessageID: "456",
			expected:      "https://desk.exp.channel.io/#/channels/1/team_chats/groups/23/456",
		},
		{
			name: "other",
			svc: &ServiceImpl{
				deskURL: "https://desk.channel.io",
			},
			group: model.Group{
				ChannelID: "9",
				ID:        "87",
			},
			rootMessageID: "654",
			expected:      "https://desk.channel.io/#/channels/9/team_chats/groups/87/654",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual := tc.svc.BuildTeamChatURL(tc.group, tc.rootMessageID)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestServiceImpl_FindManagerByGitHubMentionUsername_RefetchesOnMiss(t *testing.T) {
	t.Parallel()

	beforeRegister := []model.Manager{
		{ID: "1", Name: "Claud", GithubUsername: ptrString("ch-claud")},
	}
	afterRegister := []model.Manager{
		{ID: "1", Name: "Claud", GithubUsername: ptrString("ch-claud")},
		{ID: "2", Name: "Dylan", GithubUsername: ptrString("ch-dylan")},
	}

	m := new(mockClient)
	// 첫 조회는 등록 전 데이터, 미스 후 재조회는 등록 후 데이터를 반환한다.
	m.On("ListManagers", mock.Anything, mock.Anything).Return(beforeRegister, nil).Once()
	m.On("ListManagers", mock.Anything, mock.Anything).Return(afterRegister, nil).Once()

	s := NewServiceImpl(m, new(config.Config))

	actual, err := s.FindManagerByGitHubMentionUsername(context.TODO(), "1", "ch-dylan")
	assert.NoError(t, err)
	if assert.NotNil(t, actual) {
		assert.Equal(t, "2", actual.ID)
	}
	// 캐시 TTL 을 기다리지 않고 맵을 다시 만들어 재조회했는지 확인한다.
	m.AssertNumberOfCalls(t, "ListManagers", 2)
}

func TestServiceImpl_FindManagerByGitHubMentionUsername_CacheHitDoesNotRefetch(t *testing.T) {
	t.Parallel()

	managers := []model.Manager{
		{ID: "1", Name: "Claud", GithubUsername: ptrString("ch-claud")},
	}

	m := new(mockClient)
	m.On("ListManagers", mock.Anything, mock.Anything).Return(managers, nil)

	s := NewServiceImpl(m, new(config.Config))

	ctx := context.TODO()
	for range 3 {
		actual, err := s.FindManagerByGitHubMentionUsername(ctx, "1", "ch-claud")
		assert.NoError(t, err)
		if assert.NotNil(t, actual) {
			assert.Equal(t, "1", actual.ID)
		}
	}
	// 맵에 있는 username 은 재조회를 유발하지 않는다.
	m.AssertNumberOfCalls(t, "ListManagers", 1)
}

func TestServiceImpl_FindManagerByGitHubMentionUsername_RefetchesOncePerMiss(t *testing.T) {
	t.Parallel()

	managers := []model.Manager{
		{ID: "1", Name: "Claud", GithubUsername: ptrString("ch-claud")},
	}

	m := new(mockClient)
	m.On("ListManagers", mock.Anything, mock.Anything).Return(managers, nil)

	s := NewServiceImpl(m, new(config.Config))

	ctx := context.TODO()
	// 첫 호출: 캐시 생성 1회 + 미스 재조회 1회
	actual, err := s.FindManagerByGitHubMentionUsername(ctx, "1", "outside-contributor")
	assert.NoError(t, err)
	assert.Nil(t, actual)
	m.AssertNumberOfCalls(t, "ListManagers", 2)

	// 이후 호출도 미스마다 재조회를 1회씩만 수행한다. (호출당 최대 1회)
	for range 3 {
		actual, err = s.FindManagerByGitHubMentionUsername(ctx, "1", "outside-contributor")
		assert.NoError(t, err)
		assert.Nil(t, actual)
	}
	m.AssertNumberOfCalls(t, "ListManagers", 5)
}

type mockClient struct {
	mock.Mock
	client.Client
}

func (m *mockClient) ListManagers(ctx context.Context, channelID string) ([]model.Manager, error) {
	args := m.Called(ctx, channelID)
	return args.Get(0).([]model.Manager), args.Error(1)
}

type mockManagerCache struct {
	mock.Mock
}

func (m *mockManagerCache) Get(ctx context.Context, key string) (*map[string]model.Manager, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(*map[string]model.Manager), args.Error(1)
}
func (m *mockManagerCache) Set(
	ctx context.Context,
	key string,
	value map[string]model.Manager,
	expiry time.Duration,
) error {
	args := m.Called(ctx, key, value, expiry)
	return args.Error(0)
}

func ptrString(s string) *string {
	return &s
}
