package forges

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v82/github"
)

func TestGitHubFetchRepository(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(github.Repository{
			FullName:         ptr("octocat/hello-world"),
			Name:             ptr("hello-world"),
			Description:      ptr("My first repository"),
			Homepage:         ptr("https://example.com"),
			HTMLURL:          ptr("https://github.com/octocat/hello-world"),
			Language:         ptr("Go"),
			DefaultBranch:    ptr("main"),
			Fork:             ptrBool(false),
			Archived:         ptrBool(false),
			Private:          ptrBool(false),
			MirrorURL:        ptr(""),
			Size:             ptrInt(1024),
			StargazersCount:  ptrInt(100),
			ForksCount:       ptrInt(50),
			OpenIssuesCount:  ptrInt(10),
			SubscribersCount: ptrInt(25),
			HasIssues:        ptrBool(true),
			Topics:           []string{"go", "cli"},
			Owner: &github.User{
				Login:     ptr("octocat"),
				AvatarURL: ptr("https://avatars.githubusercontent.com/u/1?v=4"),
			},
			License: &github.License{
				SPDXID: ptr("MIT"),
			},
			Parent: &github.Repository{
				FullName: ptr("upstream/hello-world"),
			},
			CreatedAt: &github.Timestamp{Time: parseTime("2020-01-01T00:00:00Z")},
			UpdatedAt: &github.Timestamp{Time: parseTime("2024-06-15T12:00:00Z")},
			PushedAt:  &github.Timestamp{Time: parseTime("2024-06-15T11:00:00Z")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	repo, err := f.FetchRepository(context.Background(), "octocat", "hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEqual(t, "FullName", "octocat/hello-world", repo.FullName)
	assertEqual(t, "Owner", "octocat", repo.Owner)
	assertEqual(t, "Name", "hello-world", repo.Name)
	assertEqual(t, "Description", "My first repository", repo.Description)
	assertEqual(t, "Homepage", "https://example.com", repo.Homepage)
	assertEqual(t, "HTMLURL", "https://github.com/octocat/hello-world", repo.HTMLURL)
	assertEqual(t, "Language", "Go", repo.Language)
	assertEqual(t, "License", "MIT", repo.License)
	assertEqual(t, "DefaultBranch", "main", repo.DefaultBranch)
	assertEqualBool(t, "Fork", false, repo.Fork)
	assertEqualBool(t, "Archived", false, repo.Archived)
	assertEqualBool(t, "Private", false, repo.Private)
	assertEqualInt(t, "Size", 1024, repo.Size)
	assertEqualInt(t, "StargazersCount", 100, repo.StargazersCount)
	assertEqualInt(t, "ForksCount", 50, repo.ForksCount)
	assertEqualInt(t, "OpenIssuesCount", 10, repo.OpenIssuesCount)
	assertEqualInt(t, "SubscribersCount", 25, repo.SubscribersCount)
	assertEqualBool(t, "HasIssues", true, repo.HasIssues)
	assertEqualBool(t, "PullRequestsEnabled", true, repo.PullRequestsEnabled)
	assertEqual(t, "SourceName", "upstream/hello-world", repo.SourceName)
	assertEqual(t, "LogoURL", "https://avatars.githubusercontent.com/u/1?v=4", repo.LogoURL)
	assertSliceEqual(t, "Topics", []string{"go", "cli"}, repo.Topics)
}

func TestGitHubFetchRepositoryNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	_, err := f.FetchRepository(context.Background(), "octocat", "nonexistent")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGitHubFetchRepositoryNoassertionLicense(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/test/noassertion", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(github.Repository{
			FullName: ptr("test/noassertion"),
			Name:     ptr("noassertion"),
			Owner:    &github.User{Login: ptr("test")},
			License:  &github.License{SPDXID: ptr("NOASSERTION")},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	repo, err := f.FetchRepository(context.Background(), "test", "noassertion")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.License != "" {
		t.Errorf("expected empty license for NOASSERTION, got %q", repo.License)
	}
}

func TestGitHubListRepositories(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/orgs/myorg/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]*github.Repository{
			{
				FullName: ptr("myorg/repo-a"),
				Name:     ptr("repo-a"),
				Owner:    &github.User{Login: ptr("myorg")},
				Language: ptr("Go"),
				Archived: ptrBool(false),
				Fork:     ptrBool(false),
			},
			{
				FullName: ptr("myorg/repo-b"),
				Name:     ptr("repo-b"),
				Owner:    &github.User{Login: ptr("myorg")},
				Language: ptr("Rust"),
				Archived: ptrBool(true),
				Fork:     ptrBool(false),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	repos, err := f.ListRepositories(context.Background(), "myorg", ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "myorg/repo-a", repos[0].FullName)
	assertEqual(t, "repos[1].FullName", "myorg/repo-b", repos[1].FullName)
}

func TestGitHubListRepositoriesFallbackToUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/orgs/someuser/repos", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	})
	mux.HandleFunc("GET /api/v3/users/someuser/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]*github.Repository{
			{
				FullName: ptr("someuser/personal"),
				Name:     ptr("personal"),
				Owner:    &github.User{Login: ptr("someuser")},
				Fork:     ptrBool(false),
				Archived: ptrBool(false),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	repos, err := f.ListRepositories(context.Background(), "someuser", ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "someuser/personal", repos[0].FullName)
}

func TestGitHubListRepositoriesWithFilters(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/orgs/myorg/repos", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]*github.Repository{
			{
				FullName: ptr("myorg/active"),
				Name:     ptr("active"),
				Owner:    &github.User{Login: ptr("myorg")},
				Archived: ptrBool(false),
				Fork:     ptrBool(false),
			},
			{
				FullName: ptr("myorg/archived"),
				Name:     ptr("archived"),
				Owner:    &github.User{Login: ptr("myorg")},
				Archived: ptrBool(true),
				Fork:     ptrBool(false),
			},
			{
				FullName: ptr("myorg/fork"),
				Name:     ptr("fork"),
				Owner:    &github.User{Login: ptr("myorg")},
				Archived: ptrBool(false),
				Fork:     ptrBool(true),
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	repos, err := f.ListRepositories(context.Background(), "myorg", ListOptions{
		Archived: ArchivedExclude,
		Forks:    ForkExclude,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	assertEqual(t, "repos[0].FullName", "myorg/active", repos[0].FullName)
}

func TestGitHubFetchTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/repos/octocat/hello-world/tags", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]github.RepositoryTag{
			{
				Name:   ptr("v1.0.0"),
				Commit: &github.Commit{SHA: ptr("abc123")},
			},
			{
				Name:   ptr("v0.9.0"),
				Commit: &github.Commit{SHA: ptr("def456")},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := github.NewClient(nil)
	c, _ = c.WithEnterpriseURLs(srv.URL+"/api/v3", srv.URL+"/api/v3")
	f := &gitHubForge{client: c}

	tags, err := f.FetchTags(context.Background(), "octocat", "hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	assertEqual(t, "Tag[0].Name", "v1.0.0", tags[0].Name)
	assertEqual(t, "Tag[0].Commit", "abc123", tags[0].Commit)
	assertEqual(t, "Tag[1].Name", "v0.9.0", tags[1].Name)
	assertEqual(t, "Tag[1].Commit", "def456", tags[1].Commit)
}
