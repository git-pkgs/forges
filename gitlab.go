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

func convertGitLabProject(p *gitlab.Project) Repository {
	result := Repository{
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

	return result
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

	result := convertGitLabProject(p)
	return &result, nil
}

func (f *gitLabForge) ListRepositories(ctx context.Context, owner string, opts ListOptions) ([]Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	// Try group endpoint first, fall back to user projects on 404.
	repos, err := f.listGroupProjects(ctx, owner, perPage)
	if err != nil {
		repos, err = f.listUserProjects(ctx, owner, perPage)
		if err != nil {
			return nil, err
		}
	}

	return FilterRepos(repos, opts), nil
}

func (f *gitLabForge) listGroupProjects(ctx context.Context, group string, perPage int) ([]Repository, error) {
	var all []Repository
	glOpts := &gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage)},
	}
	for {
		projects, resp, err := f.client.Groups.ListGroupProjects(group, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, p := range projects {
			all = append(all, convertGitLabProject(p))
		}
		if resp.NextPage == 0 {
			break
		}
		glOpts.Page = resp.NextPage
	}
	return all, nil
}

func (f *gitLabForge) listUserProjects(ctx context.Context, user string, perPage int) ([]Repository, error) {
	var all []Repository
	glOpts := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{PerPage: int64(perPage)},
	}
	for {
		projects, resp, err := f.client.Projects.ListUserProjects(user, glOpts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, p := range projects {
			all = append(all, convertGitLabProject(p))
		}
		if resp.NextPage == 0 {
			break
		}
		glOpts.Page = resp.NextPage
	}
	return all, nil
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
