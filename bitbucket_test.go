package forges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBitbucketFetchRepository(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/stash-example-plugin", func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-bb-token" {
			t.Errorf("expected Bearer token, got %q", auth)
		}
		json.NewEncoder(w).Encode(bbRepository{
			Slug:        "stash-example-plugin",
			Name:        "stash-example-plugin",
			FullName:    "atlassian/stash-example-plugin",
			Description: "An example Bitbucket plugin",
			Website:     "https://example.atlassian.com",
			Language:    "java",
			IsPrivate:   false,
			Size:        256,
			HasIssues:   true,
			MainBranch: &struct {
				Name string `json:"name"`
			}{Name: "master"},
			Owner: &struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
			}{Username: "atlassian", DisplayName: "Atlassian"},
			Parent: &struct {
				FullName string `json:"full_name"`
			}{FullName: "original/stash-example-plugin"},
			Links: struct {
				HTML struct {
					Href string `json:"href"`
				} `json:"html"`
				Avatar struct {
					Href string `json:"href"`
				} `json:"avatar"`
			}{
				HTML: struct {
					Href string `json:"href"`
				}{Href: "https://bitbucket.org/atlassian/stash-example-plugin"},
				Avatar: struct {
					Href string `json:"href"`
				}{Href: "https://bitbucket.org/atlassian/stash-example-plugin/avatar"},
			},
			CreatedOn: "2013-10-01T18:35:13.270530+00:00",
			UpdatedOn: "2024-01-15T09:22:00.000000+00:00",
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Override the bitbucket API base URL
	origAPI := bitbucketAPI
	defer func() { setBitbucketAPI(origAPI) }()
	setBitbucketAPI(srv.URL + "/2.0")

	f := newBitbucketForge("test-bb-token", nil)

	repo, err := f.FetchRepository(context.Background(), "atlassian", "stash-example-plugin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "atlassian/stash-example-plugin", repo.FullName)
	assertEqual(t, "Owner", "atlassian", repo.Owner)
	assertEqual(t, "Name", "stash-example-plugin", repo.Name)
	assertEqual(t, "Description", "An example Bitbucket plugin", repo.Description)
	assertEqual(t, "Homepage", "https://example.atlassian.com", repo.Homepage)
	assertEqual(t, "HTMLURL", "https://bitbucket.org/atlassian/stash-example-plugin", repo.HTMLURL)
	assertEqual(t, "Language", "java", repo.Language)
	assertEqual(t, "DefaultBranch", "master", repo.DefaultBranch)
	assertEqualBool(t, "Private", false, repo.Private)
	assertEqualBool(t, "Fork", true, repo.Fork)
	assertEqual(t, "SourceName", "original/stash-example-plugin", repo.SourceName)
	assertEqual(t, "LogoURL", "https://bitbucket.org/atlassian/stash-example-plugin/avatar", repo.LogoURL)
	assertEqualInt(t, "Size", 256, repo.Size)
	assertEqualBool(t, "HasIssues", true, repo.HasIssues)
}

func TestBitbucketFetchRepositoryNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origAPI := bitbucketAPI
	defer func() { setBitbucketAPI(origAPI) }()
	setBitbucketAPI(srv.URL + "/2.0")

	f := newBitbucketForge("", nil)

	_, err := f.FetchRepository(context.Background(), "atlassian", "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBitbucketListRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"values": []bbRepository{
				{
					Slug:     "repo-a",
					FullName: "atlassian/repo-a",
					Language: "java",
					Owner: &struct {
						Username    string `json:"username"`
						DisplayName string `json:"display_name"`
					}{Username: "atlassian"},
				},
				{
					Slug:     "repo-b",
					FullName: "atlassian/repo-b",
					Language: "python",
					Owner: &struct {
						Username    string `json:"username"`
						DisplayName string `json:"display_name"`
					}{Username: "atlassian"},
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origAPI := bitbucketAPI
	defer func() { setBitbucketAPI(origAPI) }()
	setBitbucketAPI(srv.URL + "/2.0")

	f := newBitbucketForge("test-token", nil)

	repos, err := f.ListRepositories(context.Background(), "atlassian", ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "atlassian/repo-a", repos[0].FullName)
	assertEqual(t, "repos[1].FullName", "atlassian/repo-b", repos[1].FullName)
}

func TestBitbucketListRepositoriesNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origAPI := bitbucketAPI
	defer func() { setBitbucketAPI(origAPI) }()
	setBitbucketAPI(srv.URL + "/2.0")

	f := newBitbucketForge("", nil)

	_, err := f.ListRepositories(context.Background(), "nonexistent", ListOptions{})
	if err != ErrOwnerNotFound {
		t.Fatalf("expected ErrOwnerNotFound, got %v", err)
	}
}

func TestBitbucketFetchTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /2.0/repositories/atlassian/myrepo/refs/tags", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(bbTagsResponse{
			Values: []bbTag{
				{Name: "v1.0.0", Target: struct {
					Hash string `json:"hash"`
				}{Hash: "eee555"}},
				{Name: "v0.1.0", Target: struct {
					Hash string `json:"hash"`
				}{Hash: "fff666"}},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	origAPI := bitbucketAPI
	defer func() { setBitbucketAPI(origAPI) }()
	setBitbucketAPI(srv.URL + "/2.0")

	f := newBitbucketForge("", nil)

	tags, err := f.FetchTags(context.Background(), "atlassian", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	assertEqual(t, "Tag[0].Name", "v1.0.0", tags[0].Name)
	assertEqual(t, "Tag[0].Commit", "eee555", tags[0].Commit)
	assertEqual(t, "Tag[1].Name", "v0.1.0", tags[1].Name)
	assertEqual(t, "Tag[1].Commit", "fff666", tags[1].Commit)
}
