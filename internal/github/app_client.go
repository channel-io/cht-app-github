package github

import (
	"context"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v60/github"
)

// AppClient NOTE : https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/about-authentication-with-a-github-app
// installation 정보 등 앱 자체에 대한 정보를 활용할때 사용합니다.
type AppClient struct {
	*github.Client
	metrics *ClientMetrics
}

func newAppClient(appID int64, privateKey []byte, metrics *ClientMetrics) (*AppClient, error) {
	transport, err := ghinstallation.NewAppsTransport(http.DefaultTransport, appID, privateKey)
	if err != nil {
		return nil, err
	}
	return &AppClient{
		Client:  github.NewClient(&http.Client{Transport: transport}),
		metrics: metrics,
	}, nil
}

func (c *AppClient) ListInstallations(ctx context.Context) ([]*github.Installation, error) {
	var results []*github.Installation
	nextPage := 1
	for {
		installations, response, err := c.Apps.ListInstallations(ctx, &github.ListOptions{
			Page: nextPage,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, installations...)

		nextPage = response.NextPage
		if nextPage == 0 {
			break
		}
	}
	return results, nil
}
