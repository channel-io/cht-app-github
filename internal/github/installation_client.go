package github

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/go-errors/errors"
	"github.com/google/go-github/v60/github"
)

// InstallationClient TODO : rate limit 초과 에 대한 대응
type InstallationClient struct {
	*github.Client
	installationContext InstallationContext
	metrics             *ClientMetrics
}

func newInstallationClient(
	githubClient *github.Client,
	eventContext InstallationContext,
	metrics *ClientMetrics,
) *InstallationClient {
	return &InstallationClient{
		Client:              githubClient,
		installationContext: eventContext,
		metrics:             metrics,
	}
}

func newGithubClientWithInstallation(appID int64, privateKey []byte, context InstallationContext) (*github.Client, error) {
	transport, err := ghinstallation.New(
		http.DefaultTransport,
		appID,
		context.InstallationId,
		privateKey,
	)
	if err != nil {
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: transport}), nil
}

func (c *InstallationClient) CreateCommentOnIssue(ctx context.Context, org, repo string, number int, body string) error {
	_, res, err := c.Issues.CreateComment(ctx, org, repo, number, &github.IssueComment{
		Body: &body,
	})

	if err != nil {
		return err
	}
	c.metrics.onResponse(c.installationContext, "issue.create_comment", res, err)

	return nil
}

func (c *InstallationClient) FindCustomProperty(ctx context.Context, repository, key string) (string, error) {
	values, res, err := c.Repositories.GetAllCustomPropertyValues(ctx, c.installationContext.OrgLogin, repository)
	if err != nil {
		return "", err
	}
	c.metrics.onResponse(c.installationContext, "repo.get_all_custom_property_values", res, err)

	for _, value := range values {
		if value.PropertyName == key {
			return *value.Value, nil
		}
	}
	return "", errors.Errorf("%s custom property required in Org(%s) Repository(%s)", key, c.installationContext.OrgLogin, repository)
}

// TODO @Dylan : list order 재확인 필요.
func (c *InstallationClient) FindAllCommentsOnIssue(ctx context.Context, repository string, number int) ([]*github.IssueComment, error) {
	comments, res, err := c.Issues.ListComments(ctx, c.installationContext.OrgLogin, repository, number, nil)
	if err != nil {
		return nil, err
	}
	c.metrics.onResponse(c.installationContext, "issue.list_comments", res, err)

	return comments, nil
}

func (c *InstallationClient) AddAssigneeToIssue(ctx context.Context, repository string, number int, assignees []string) error {
	var assigneeCandidates []string
	// Checks if a user has permission to be assigned to an issue in this repository.
	for _, assignee := range assignees {
		_, res, err := c.Issues.IsAssignee(ctx, c.installationContext.OrgLogin, repository, assignee)
		if err != nil {
			return err
		}
		if res.StatusCode == 404 {
			log.Printf("%s user does not have permission to be assigned in Org(%s) Repository(%s)\n", assignee, c.installationContext.OrgLogin, repository)
			continue
		}
		assigneeCandidates = append(assigneeCandidates, assignee)
	}

	_, res, err := c.Issues.AddAssignees(ctx, c.installationContext.OrgLogin, repository, number, assigneeCandidates)
	if err != nil {
		return err
	}
	c.metrics.onResponse(c.installationContext, "issue.add_assignee", res, err)

	return nil
}

func (c *InstallationClient) ListReviewRequestedPullRequests(ctx context.Context, user string) ([]*github.Issue, error) {
	query := fmt.Sprintf("type:pr state:open org:%s review-requested:%s", c.installationContext.OrgLogin, user)
	result, response, err := c.Search.Issues(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	c.metrics.onResponse(c.installationContext, "org.search_pull_requests", response, err)
	return result.Issues, nil
}

func (c *InstallationClient) ListOpenedPullRequests(ctx context.Context, user string) ([]*github.Issue, error) {
	query := fmt.Sprintf("type:pr state:open org:%s assignee:%s", c.installationContext.OrgLogin, user)
	result, response, err := c.Search.Issues(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	c.metrics.onResponse(c.installationContext, "org.search_pull_requests", response, err)
	return result.Issues, nil
}
