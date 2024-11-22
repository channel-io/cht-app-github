package github

import "github.com/google/go-github/v60/github"

type FilterPullRequestPredicate func(pr *github.PullRequest) bool

func WithClosedPullRequestFilter() FilterPullRequestPredicate {
	return func(pr *github.PullRequest) bool {
		return pr.GetState() == "closed"
	}
}

func WithNotDraftPullRequestFilter() FilterPullRequestPredicate {
	return func(pr *github.PullRequest) bool {
		return !pr.GetDraft()
	}
}

func WithMergedPullRequestFilter() FilterPullRequestPredicate {
	return func(pr *github.PullRequest) bool {
		// NOTE : merged PR 라고 하더라도 api 에 따라 merged 필드가 nil 로 설정되는 경우가 있음.
		// List api 에 대해서는 merged 필드가 nil 로 내려오면서 동시에 mergedAt 은 값이 정상적으로 설정됨.
		// 반면, Get api 에서는 merged 필드와 mergedAt 필드가 모두 정상적으로 설정되어 있음.
		// 아래에서는 mergedAt을 우선 판단함.
		return pr.MergedAt != nil || (pr.Merged != nil && pr.GetMerged())
	}
}
