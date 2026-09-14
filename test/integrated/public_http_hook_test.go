package integrated

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/channel-io/cht-app-github/api/public"
	channelclient "github.com/channel-io/cht-app-github/internal/channel/client"
	"github.com/channel-io/cht-app-github/internal/channel/model"
	"github.com/channel-io/cht-app-github/internal/github"
	libhttp "github.com/channel-io/cht-app-github/internal/http"
	"github.com/channel-io/cht-app-github/tool"
)

// example Github nickname: ski-channel
// example Channel id: ski
// ski-channel -> ski
const pullRequestReadyForReviewPayload = `{
  "action": "ready_for_review",
  "number": 42,
  "pull_request": {
    "number": 42,
    "title": "test pull request",
    "html_url": "https://github.com/channel-io/cht-app-github/pull/42",
    "draft": false,
    "state": "open",
    "user": { "login": "someone", "type": "User" },
    "requested_reviewers": [{ "login": "ski-channel", "type": "User" }],
    "base": { "ref": "main" },
    "head": { "ref": "feat/test" }
  },
  "repository": { "name": "cht-app-github", "full_name": "channel-io/cht-app-github" },
  "organization": { "login": "channel-io" },
  "installation": { "id": 12345 },
  "sender": { "login": "someone", "type": "User" }
}`

type stubGithubService struct {
	github.Service
	comments []string
}

func (s *stubGithubService) FindGroup(context.Context, github.InstallationContext, string) (model.Group, error) {
	return model.Group{ChannelID: "1", ID: "544964"}, nil
}

func (s *stubGithubService) FindRootMessageID(
	context.Context, github.InstallationContext, string, int, int,
) (*string, error) {
	return nil, nil
}

func (s *stubGithubService) CreateComment(
	_ context.Context, _ github.InstallationContext, _ string, _ int, body string,
) error {
	s.comments = append(s.comments, body)
	return nil
}

type stubChannelClient struct {
	channelclient.Client

	written []*model.Message
}

func (s *stubChannelClient) ListManagers(context.Context, string) ([]model.Manager, error) {
	username := "ski-channel"
	return []model.Manager{{ID: "676545", Name: "ski", GithubUsername: &username}}, nil
}

func (s *stubChannelClient) WriteGroupMessage(
	_ context.Context, _ string, _ string, message *model.Message,
) (string, error) {
	s.written = append(s.written, message)
	return "written_message_id", nil
}

func TestHookPullRequestReadyForReview(t *testing.T) {
	githubStub := &stubGithubService{}
	channelStub := &stubChannelClient{}

	tool.NewIntegratedTestSuite().
		Target(public.HTTPServerTestModule()).
		Target(integratedTestModule()).
		Mock(fx.Options(
			fx.Decorate(func(github.Service) github.Service { return githubStub }),
			fx.Decorate(func(channelclient.Client) channelclient.Client { return channelStub }),
		)).
		Test(func(server *libhttp.Server, w *httptest.ResponseRecorder) {
			req, _ := http.NewRequest("POST", "/hook/v1", bytes.NewBufferString(pullRequestReadyForReviewPayload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", "pull_request")
			req.Header.Set("X-GitHub-Delivery", "00000000-0000-0000-0000-000000000000")

			server.Serve(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			require.Len(t, channelStub.written, 1)
			require.Len(t, channelStub.written[0].Blocks, 1)
			assert.Contains(
				t,
				channelStub.written[0].Blocks[0].Text.Value,
				`<link type="manager" value="676545">ski</link>`,
			)

			require.Len(t, githubStub.comments, 1)
			t.Logf("team chat message: %s", channelStub.written[0].Blocks[0].Text.Value)
			t.Logf("github comment: %s", githubStub.comments[0])
		}).
		Run()
}
