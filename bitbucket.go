package forges

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var bitbucketAPI = "https://api.bitbucket.org/2.0"

// setBitbucketAPI overrides the Bitbucket API base URL (for testing).
func setBitbucketAPI(url string) { bitbucketAPI = url }

type bitbucketForge struct {
	token      string
	httpClient *http.Client
}

func newBitbucketForge(token string, hc *http.Client) *bitbucketForge {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &bitbucketForge{token: token, httpClient: hc}
}

// Bitbucket API response types

type bbRepository struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Language    string `json:"language"`
	IsPrivate   bool   `json:"is_private"`
	ForkPolicy  string `json:"fork_policy"`
	Size        int    `json:"size"`
	HasIssues   bool   `json:"has_issues"`
	MainBranch  *struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Owner *struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Parent *struct {
		FullName string `json:"full_name"`
	} `json:"parent"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
		Avatar struct {
			Href string `json:"href"`
		} `json:"avatar"`
	} `json:"links"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type bbTagsResponse struct {
	Values []bbTag `json:"values"`
	Next   string  `json:"next"`
}

type bbTag struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
	} `json:"target"`
}

func (f *bitbucketForge) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if f.token != "" {
		req.Header.Set("Authorization", "Bearer "+f.token)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &HTTPError{StatusCode: resp.StatusCode, URL: url, Body: string(body)}
	}

	return json.NewDecoder(resp.Body).Decode(v)
}

func (f *bitbucketForge) FetchRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repositories/%s/%s", bitbucketAPI, owner, repo)
	var bb bbRepository
	if err := f.getJSON(ctx, url, &bb); err != nil {
		return nil, err
	}

	result := &Repository{
		FullName:    bb.FullName,
		Name:        bb.Slug,
		Description: bb.Description,
		Homepage:    bb.Website,
		Language:    bb.Language,
		Private:     bb.IsPrivate,
		Size:        bb.Size,
		HasIssues:   bb.HasIssues,
		HTMLURL:     bb.Links.HTML.Href,
		LogoURL:     bb.Links.Avatar.Href,
	}

	if bb.Owner != nil {
		result.Owner = bb.Owner.Username
	}

	if bb.MainBranch != nil {
		result.DefaultBranch = bb.MainBranch.Name
	}

	if bb.Parent != nil {
		result.Fork = true
		result.SourceName = bb.Parent.FullName
	}

	if t, err := time.Parse(time.RFC3339, bb.CreatedOn); err == nil {
		result.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, bb.UpdatedOn); err == nil {
		result.UpdatedAt = t
	}

	return result, nil
}

func (f *bitbucketForge) FetchTags(ctx context.Context, owner, repo string) ([]Tag, error) {
	var allTags []Tag
	url := fmt.Sprintf("%s/repositories/%s/%s/refs/tags?pagelen=100", bitbucketAPI, owner, repo)

	for url != "" {
		var page bbTagsResponse
		if err := f.getJSON(ctx, url, &page); err != nil {
			return nil, err
		}
		for _, t := range page.Values {
			allTags = append(allTags, Tag{
				Name:   t.Name,
				Commit: t.Target.Hash,
			})
		}
		url = page.Next
	}
	return allTags, nil
}
