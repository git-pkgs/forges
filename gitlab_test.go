package forges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitLabFetchRepository(t *testing.T) {
	created := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	lastActivity := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	mux := http.NewServeMux()
	// GitLab SDK URL-encodes the project path: mygroup/myrepo -> mygroup%2Fmyrepo
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"path_with_namespace":    "mygroup/myrepo",
			"name":                   "myrepo",
			"description":            "A GitLab project",
			"web_url":                "https://gitlab.com/mygroup/myrepo",
			"default_branch":         "main",
			"archived":               false,
			"visibility":             "public",
			"star_count":             42,
			"forks_count":            7,
			"open_issues_count":      3,
			"merge_requests_enabled": true,
			"topics":                 []string{"rust", "wasm"},
			"namespace": map[string]any{
				"path":       "mygroup",
				"avatar_url": "https://gitlab.com/uploads/-/system/group/avatar/123/logo.png",
			},
			"license": map[string]any{
				"key":  "apache-2.0",
				"name": "Apache License 2.0",
			},
			"forked_from_project": map[string]any{
				"path_with_namespace": "upstream/myrepo",
			},
			"created_at":       created.Format(time.RFC3339),
			"last_activity_at": lastActivity.Format(time.RFC3339),
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGitLabForge(srv.URL, "test-token", nil)

	repo, err := f.FetchRepository(context.Background(), "mygroup", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "mygroup/myrepo", repo.FullName)
	assertEqual(t, "Owner", "mygroup", repo.Owner)
	assertEqual(t, "Name", "myrepo", repo.Name)
	assertEqual(t, "Description", "A GitLab project", repo.Description)
	assertEqual(t, "HTMLURL", "https://gitlab.com/mygroup/myrepo", repo.HTMLURL)
	assertEqual(t, "DefaultBranch", "main", repo.DefaultBranch)
	assertEqualBool(t, "Archived", false, repo.Archived)
	assertEqualBool(t, "Private", false, repo.Private)
	assertEqualInt(t, "StargazersCount", 42, repo.StargazersCount)
	assertEqualInt(t, "ForksCount", 7, repo.ForksCount)
	assertEqualInt(t, "OpenIssuesCount", 3, repo.OpenIssuesCount)
	assertEqualBool(t, "PullRequestsEnabled", true, repo.PullRequestsEnabled)
	assertEqual(t, "License", "apache-2.0", repo.License)
	assertEqualBool(t, "Fork", true, repo.Fork)
	assertEqual(t, "SourceName", "upstream/myrepo", repo.SourceName)
	assertEqual(t, "LogoURL", "https://gitlab.com/uploads/-/system/group/avatar/123/logo.png", repo.LogoURL)
	assertSliceEqual(t, "Topics", []string{"rust", "wasm"}, repo.Topics)
}

func TestGitLabFetchRepositoryNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fnonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGitLabForge(srv.URL, "", nil)

	_, err := f.FetchRepository(context.Background(), "mygroup", "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitLabFetchTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v4/projects/mygroup%2Fmyrepo/repository/tags", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"name":   "v2.0.0",
				"commit": map[string]string{"id": "aaa111"},
			},
			{
				"name":   "v1.0.0",
				"commit": map[string]string{"id": "bbb222"},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := newGitLabForge(srv.URL, "", nil)

	tags, err := f.FetchTags(context.Background(), "mygroup", "myrepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	assertEqual(t, "Tag[0].Name", "v2.0.0", tags[0].Name)
	assertEqual(t, "Tag[0].Commit", "aaa111", tags[0].Commit)
	assertEqual(t, "Tag[1].Name", "v1.0.0", tags[1].Name)
	assertEqual(t, "Tag[1].Commit", "bbb222", tags[1].Commit)
}
