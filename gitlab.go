package forges

import (
	"context"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type gitLabForge struct {
	client *gitlab.Client
}

func newGitLabForge(baseURL, token string, hc *http.Client) *gitLabForge {
	opts := []gitlab.ClientOptionFunc{
		gitlab.WithBaseURL(baseURL + "/api/v4"),
	}
	if hc != nil {
		opts = append(opts, gitlab.WithHTTPClient(hc))
	}
	c, _ := gitlab.NewClient(token, opts...)
	return &gitLabForge{client: c}
}

func (f *gitLabForge) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	pid := owner + "/" + repo
	license := true
	p, resp, err := f.client.Projects.GetProject(pid, &gitlab.GetProjectOptions{
		License: &license,
	})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := &Repository{
		FullName:            p.PathWithNamespace,
		Name:                p.Name,
		Description:         p.Description,
		HTMLURL:             p.WebURL,
		DefaultBranch:       p.DefaultBranch,
		Archived:            p.Archived,
		Private:             p.Visibility == gitlab.PrivateVisibility,
		StargazersCount:     int(p.StarCount),
		ForksCount:          int(p.ForksCount),
		OpenIssuesCount:     int(p.OpenIssuesCount),
		HasIssues:           true,
		PullRequestsEnabled: p.MergeRequestsEnabled,
		Topics:              p.Topics,
	}

	if p.Namespace != nil {
		result.Owner = p.Namespace.Path
		result.LogoURL = p.Namespace.AvatarURL
	}

	if p.License != nil {
		result.License = p.License.Key
	}

	if p.ForkedFromProject != nil {
		result.Fork = true
		result.SourceName = p.ForkedFromProject.PathWithNamespace
	}

	if p.CreatedAt != nil {
		result.CreatedAt = *p.CreatedAt
	}
	if p.LastActivityAt != nil {
		result.UpdatedAt = *p.LastActivityAt
	}

	return result, nil
}

func (f *gitLabForge) FetchTags(ctx context.Context, owner, repo string) ([]Tag, error) {
	pid := owner + "/" + repo
	var allTags []Tag
	opts := &gitlab.ListTagsOptions{
		ListOptions: gitlab.ListOptions{PerPage: 100},
	}
	for {
		tags, resp, err := f.client.Tags.ListTags(pid, opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := Tag{Name: t.Name}
			if t.Commit != nil {
				tag.Commit = t.Commit.ID
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
