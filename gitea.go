package forges

import (
	"context"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

type giteaForge struct {
	client *gitea.Client
}

func newGiteaForge(baseURL, token string, hc *http.Client) *giteaForge {
	opts := []gitea.ClientOption{}
	if token != "" {
		opts = append(opts, gitea.SetToken(token))
	}
	if hc != nil {
		opts = append(opts, gitea.SetHTTPClient(hc))
	}
	c, _ := gitea.NewClient(baseURL, opts...)
	return &giteaForge{client: c}
}

func convertGiteaRepo(r *gitea.Repository) Repository {
	result := Repository{
		FullName:            r.FullName,
		Owner:               r.Owner.UserName,
		Name:                r.Name,
		Description:         r.Description,
		Homepage:            r.Website,
		HTMLURL:             r.HTMLURL,
		Language:            r.Language,
		DefaultBranch:       r.DefaultBranch,
		Fork:                r.Fork,
		Archived:            r.Archived,
		Private:             r.Private,
		Size:                int(r.Size),
		StargazersCount:     r.Stars,
		ForksCount:          r.Forks,
		OpenIssuesCount:     r.OpenIssues,
		HasIssues:           r.HasIssues,
		PullRequestsEnabled: r.HasPullRequests,
		LogoURL:             r.AvatarURL,
		CreatedAt:           r.Created,
		UpdatedAt:           r.Updated,
	}

	if r.Mirror {
		result.MirrorURL = r.OriginalURL
	}

	if r.Parent != nil {
		result.SourceName = r.Parent.FullName
	}

	return result
}

func (f *giteaForge) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	r, resp, err := f.client.GetRepo(owner, repo)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	result := convertGiteaRepo(r)

	// Fetch topics separately (not included in main repo response)
	topics, _, topicErr := f.client.ListRepoTopics(owner, repo, gitea.ListRepoTopicsOptions{})
	if topicErr == nil {
		result.Topics = topics
	}

	return &result, nil
}

func (f *giteaForge) ListRepositories(ctx context.Context, owner string, opts ListOptions) ([]Repository, error) {
	perPage := opts.PerPage
	if perPage <= 0 {
		perPage = 50
	}

	// Try org endpoint first, fall back to user on 404.
	repos, err := f.listOrgRepos(ctx, owner, perPage)
	if err != nil {
		repos, err = f.listUserRepos(ctx, owner, perPage)
		if err != nil {
			return nil, err
		}
	}

	return FilterRepos(repos, opts), nil
}

func (f *giteaForge) listOrgRepos(_ context.Context, owner string, perPage int) ([]Repository, error) {
	var all []Repository
	page := 1
	for {
		gRepos, resp, err := f.client.ListOrgRepos(owner, gitea.ListOrgReposOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range gRepos {
			all = append(all, convertGiteaRepo(r))
		}
		if len(gRepos) < perPage {
			break
		}
		page++
	}
	return all, nil
}

func (f *giteaForge) listUserRepos(_ context.Context, owner string, perPage int) ([]Repository, error) {
	var all []Repository
	page := 1
	for {
		gRepos, resp, err := f.client.ListUserRepos(owner, gitea.ListReposOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: perPage},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrOwnerNotFound
			}
			return nil, err
		}
		for _, r := range gRepos {
			all = append(all, convertGiteaRepo(r))
		}
		if len(gRepos) < perPage {
			break
		}
		page++
	}
	return all, nil
}

func (f *giteaForge) FetchTags(ctx context.Context, owner, repo string) ([]Tag, error) {
	var allTags []Tag
	page := 1
	for {
		tags, resp, err := f.client.ListRepoTags(owner, repo, gitea.ListRepoTagsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: 50},
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, err
		}
		for _, t := range tags {
			tag := Tag{Name: t.Name}
			if t.Commit != nil {
				tag.Commit = t.Commit.SHA
			}
			allTags = append(allTags, tag)
		}
		if len(tags) < 50 {
			break
		}
		page++
	}
	return allTags, nil
}
