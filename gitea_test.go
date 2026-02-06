package forges

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// giteaVersionHandler serves the /api/v1/version endpoint that the Gitea SDK
// requires during client creation.
func giteaVersionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"version":"1.21.0"}`)
}

func TestGiteaFetchRepository(t *testing.T) {
	created := time.Date(2021, 3, 15, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 5, 20, 8, 30, 0, 0, time.UTC)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"full_name":         "testorg/testrepo",
			"name":              "testrepo",
			"description":       "A Gitea repo",
			"website":           "https://testrepo.example.com",
			"html_url":          "https://codeberg.org/testorg/testrepo",
			"language":          "Python",
			"default_branch":    "develop",
			"fork":              true,
			"archived":          false,
			"private":           false,
			"mirror":            true,
			"original_url":      "https://github.com/upstream/testrepo",
			"size":              512,
			"stars_count":       30,
			"forks_count":       5,
			"open_issues_count": 2,
			"has_issues":        true,
			"has_pull_requests": true,
			"avatar_url":        "https://codeberg.org/repo-avatars/123",
			"created_at":        created.Format(time.RFC3339),
			"updated_at":        updated.Format(time.RFC3339),
			"owner": map[string]any{
				"login":      "testorg",
				"avatar_url": "https://codeberg.org/avatars/456",
			},
			"parent": map[string]any{
				"full_name": "upstream/testrepo",
			},
		})
	})
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/topics", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"topics": []string{"python", "machine-learning"},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGiteaForge(srv.URL, "test-token", nil)

	repo, err := f.FetchRepository(context.Background(), "testorg", "testrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "testorg/testrepo", repo.FullName)
	assertEqual(t, "Owner", "testorg", repo.Owner)
	assertEqual(t, "Name", "testrepo", repo.Name)
	assertEqual(t, "Description", "A Gitea repo", repo.Description)
	assertEqual(t, "Homepage", "https://testrepo.example.com", repo.Homepage)
	assertEqual(t, "HTMLURL", "https://codeberg.org/testorg/testrepo", repo.HTMLURL)
	assertEqual(t, "Language", "Python", repo.Language)
	assertEqual(t, "DefaultBranch", "develop", repo.DefaultBranch)
	assertEqualBool(t, "Fork", true, repo.Fork)
	assertEqualBool(t, "Archived", false, repo.Archived)
	assertEqualBool(t, "Private", false, repo.Private)
	assertEqual(t, "MirrorURL", "https://github.com/upstream/testrepo", repo.MirrorURL)
	assertEqualInt(t, "Size", 512, repo.Size)
	assertEqualInt(t, "StargazersCount", 30, repo.StargazersCount)
	assertEqualInt(t, "ForksCount", 5, repo.ForksCount)
	assertEqualInt(t, "OpenIssuesCount", 2, repo.OpenIssuesCount)
	assertEqualBool(t, "HasIssues", true, repo.HasIssues)
	assertEqualBool(t, "PullRequestsEnabled", true, repo.PullRequestsEnabled)
	assertEqual(t, "SourceName", "upstream/testrepo", repo.SourceName)
	assertEqual(t, "LogoURL", "https://codeberg.org/repo-avatars/123", repo.LogoURL)
	assertSliceEqual(t, "Topics", []string{"python", "machine-learning"}, repo.Topics)
}

func TestGiteaFetchRepositoryNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGiteaForge(srv.URL, "", nil)

	_, err := f.FetchRepository(context.Background(), "testorg", "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGiteaListRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/orgs/testorg/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"full_name":      "testorg/repo-a",
				"name":           "repo-a",
				"html_url":       "https://codeberg.org/testorg/repo-a",
				"default_branch": "main",
				"fork":           false,
				"archived":       false,
				"private":        false,
				"stars_count":    10,
				"owner":          map[string]any{"login": "testorg"},
				"created_at":     time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"updated_at":     time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
			{
				"full_name":      "testorg/repo-b",
				"name":           "repo-b",
				"html_url":       "https://codeberg.org/testorg/repo-b",
				"default_branch": "main",
				"fork":           true,
				"archived":       false,
				"private":        false,
				"stars_count":    5,
				"owner":          map[string]any{"login": "testorg"},
				"created_at":     time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"updated_at":     time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGiteaForge(srv.URL, "test-token", nil)

	repos, err := f.ListRepositories(context.Background(), "testorg", ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "testorg/repo-a", repos[0].FullName)
	assertEqual(t, "repos[1].FullName", "testorg/repo-b", repos[1].FullName)
}

func TestGiteaListRepositoriesFallbackToUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/orgs/someuser/repos", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v1/users/someuser/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"full_name":      "someuser/personal",
				"name":           "personal",
				"html_url":       "https://codeberg.org/someuser/personal",
				"default_branch": "main",
				"fork":           false,
				"archived":       false,
				"private":        false,
				"owner":          map[string]any{"login": "someuser"},
				"created_at":     time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
				"updated_at":     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGiteaForge(srv.URL, "", nil)

	repos, err := f.ListRepositories(context.Background(), "someuser", ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "someuser/personal", repos[0].FullName)
}

func TestGiteaFetchTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/version", giteaVersionHandler)
	mux.HandleFunc("GET /api/v1/repos/testorg/testrepo/tags", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":   "v3.0.0",
				"id":     "sha-tag-1",
				"commit": map[string]string{"sha": "ccc333"},
			},
			{
				"name":   "v2.0.0",
				"id":     "sha-tag-2",
				"commit": map[string]string{"sha": "ddd444"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGiteaForge(srv.URL, "", nil)

	tags, err := f.FetchTags(context.Background(), "testorg", "testrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	assertEqual(t, "Tag[0].Name", "v3.0.0", tags[0].Name)
	assertEqual(t, "Tag[0].Commit", "ccc333", tags[0].Commit)
	assertEqual(t, "Tag[1].Name", "v2.0.0", tags[1].Name)
	assertEqual(t, "Tag[1].Commit", "ddd444", tags[1].Commit)
}
