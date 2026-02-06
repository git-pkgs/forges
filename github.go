package forges

import (
	"context"
	"net/http"

	"github.com/google/go-github/v82/github"
)

type gitHubForge struct {
	client *github.Client
}

func newGitHubForge(token string, hc *http.Client) *gitHubForge {
	c := github.NewClient(hc)
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &gitHubForge{client: c}
}

func newGitHubForgeWithBase(baseURL, token string, hc *http.Client) *gitHubForge {
	c := github.NewClient(hc).WithAuthToken(token)
	c, _ = c.WithEnterpriseURLs(baseURL, baseURL)
	return &gitHubForge{client: c}
}

func (f *gitHubForge) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	r, resp, err := f.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &Repository{
		FullName:            r.GetFullName(),
		Owner:               r.GetOwner().GetLogin(),
		Name:                r.GetName(),
		Description:         r.GetDescription(),
		Homepage:            r.GetHomepage(),
		HTMLURL:             r.GetHTMLURL(),
		Language:            r.GetLanguage(),
		DefaultBranch:       r.GetDefaultBranch(),
		Fork:                r.GetFork(),
		Archived:            r.GetArchived(),
		Private:             r.GetPrivate(),
		MirrorURL:           r.GetMirrorURL(),
		Size:                r.GetSize(),
		StargazersCount:     r.GetStargazersCount(),
		ForksCount:          r.GetForksCount(),
		OpenIssuesCount:     r.GetOpenIssuesCount(),
		SubscribersCount:    r.GetSubscribersCount(),
		HasIssues:           r.GetHasIssues(),
		PullRequestsEnabled: true, // GitHub always has PRs enabled
		Topics:              r.Topics,
		LogoURL:             r.GetOwner().GetAvatarURL(),
	}

	if lic := r.GetLicense(); lic != nil {
		spdx := lic.GetSPDXID()
		if spdx != "" && spdx != "NOASSERTION" {
			result.License = spdx
		}
	}

	if parent := r.GetParent(); parent != nil {
		result.SourceName = parent.GetFullName()
	}

	if t := r.GetCreatedAt(); !t.IsZero() {
		result.CreatedAt = t.Time
	}
	if t := r.GetUpdatedAt(); !t.IsZero() {
		result.UpdatedAt = t.Time
	}
	if t := r.GetPushedAt(); !t.IsZero() {
		result.PushedAt = t.Time
	}

	return result, nil
}

func (f *gitHubForge) FetchTags(ctx context.Context, owner, repo string) ([]Tag, error) {
	var allTags []Tag
	opts := &github.ListOptions{PerPage: 100}
	for {
		tags, resp, err := f.client.Repositories.ListTags(ctx, owner, repo, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := Tag{Name: t.GetName()}
			if c := t.GetCommit(); c != nil {
				tag.Commit = c.GetSHA()
			}
			allTags = append(allTags, tag)
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allTags, nil
}
