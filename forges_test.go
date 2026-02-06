package forges

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test helpers

func ptr(s string) *string    { return &s }
func ptrBool(b bool) *bool    { return &b }
func ptrInt(i int) *int       { return &i }

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func assertEqual(t *testing.T, field, want, got string) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %q, got %q", field, want, got)
	}
}

func assertEqualBool(t *testing.T, field string, want, got bool) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %v, got %v", field, want, got)
	}
}

func assertEqualInt(t *testing.T, field string, want, got int) {
	t.Helper()
	if want != got {
		t.Errorf("%s: want %d, got %d", field, want, got)
	}
}

func assertSliceEqual(t *testing.T, field string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: want %v, got %v", field, want, got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: want %q, got %q", field, i, want[i], got[i])
		}
	}
}

// ParseRepoURL tests

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		input              string
		domain, owner, repo string
		wantErr            bool
	}{
		{
			input:  "https://github.com/octocat/hello-world",
			domain: "github.com", owner: "octocat", repo: "hello-world",
		},
		{
			input:  "https://github.com/octocat/hello-world.git",
			domain: "github.com", owner: "octocat", repo: "hello-world",
		},
		{
			input:  "https://gitlab.com/group/project/tree/main",
			domain: "gitlab.com", owner: "group", repo: "project",
		},
		{
			input:  "github.com/user/repo",
			domain: "github.com", owner: "user", repo: "repo",
		},
		{
			input:  "git@github.com:user/repo.git",
			domain: "github.com", owner: "user", repo: "repo",
		},
		{
			input:  "git@gitlab.com:group/project.git",
			domain: "gitlab.com", owner: "group", repo: "project",
		},
		{
			input:  "https://bitbucket.org/atlassian/stash-example-plugin",
			domain: "bitbucket.org", owner: "atlassian", repo: "stash-example-plugin",
		},
		{
			input:   "",
			wantErr: true,
		},
		{
			input:   "https://github.com/just-owner",
			wantErr: true,
		},
		{
			input:   "git@github.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			domain, owner, repo, err := ParseRepoURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertEqual(t, "domain", tt.domain, domain)
			assertEqual(t, "owner", tt.owner, owner)
			assertEqual(t, "repo", tt.repo, repo)
		})
	}
}

// Client routing tests

func TestClientRouting(t *testing.T) {
	c := NewClient()

	// Verify default domains are registered
	for _, domain := range []string{"github.com", "gitlab.com", "codeberg.org", "bitbucket.org"} {
		if _, err := c.forgeFor(domain); err != nil {
			t.Errorf("expected forge for %s, got error: %v", domain, err)
		}
	}

	// Unregistered domain returns error
	_, err := c.forgeFor("example.com")
	if err == nil {
		t.Error("expected error for unregistered domain")
	}
}

func TestClientFetchRepositoryRoutes(t *testing.T) {
	// Create a mock forge that records calls
	mock := &mockForge{
		repo: &Repository{FullName: "test/repo"},
	}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	repo, err := c.FetchRepository(context.Background(), "https://example.com/test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.FullName != "test/repo" {
		t.Errorf("expected test/repo, got %s", repo.FullName)
	}
	if mock.lastOwner != "test" || mock.lastRepo != "repo" {
		t.Errorf("expected owner=test repo=repo, got owner=%s repo=%s", mock.lastOwner, mock.lastRepo)
	}
}

func TestClientFetchTagsRoutes(t *testing.T) {
	mock := &mockForge{
		tags: []Tag{{Name: "v1.0.0", Commit: "abc"}},
	}
	c := &Client{
		forges: map[string]Forge{"example.com": mock},
		tokens: make(map[string]string),
	}

	tags, err := c.FetchTags(context.Background(), "https://example.com/test/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(tags))
	}
}

// Detection tests

func TestDetectForgeTypeHeaders(t *testing.T) {
	tests := []struct {
		header string
		value  string
		want   ForgeType
	}{
		{"X-GitHub-Request-Id", "abc123", GitHub},
		{"X-Gitlab-Meta", `{"cors":"abc"}`, GitLab},
		{"X-Gitea-Version", "1.21.0", Gitea},
		{"X-Forgejo-Version", "7.0.0", Forgejo},
	}

	for _, tt := range tests {
		t.Run(string(tt.want), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set(tt.header, tt.value)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			// We need to override the URL scheme, so test detectFromHeaders directly
			ft, err := detectFromHeaders(context.Background(), srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ft != tt.want {
				t.Errorf("want %s, got %s", tt.want, ft)
			}
		})
	}
}

func TestDetectForgeTypeGiteaAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"1.21.0"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != Gitea {
		t.Errorf("want Gitea, got %s", ft)
	}
}

func TestDetectForgeTypeForgejoAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"7.0.0+forgejo"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != Forgejo {
		t.Errorf("want Forgejo, got %s", ft)
	}
}

func TestDetectForgeTypeGitLabAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"16.0.0"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != GitLab {
		t.Errorf("want GitLab, got %s", ft)
	}
}

func TestDetectForgeTypeGitHubAPI(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v4/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v3/meta", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"verifiable_password_authentication": true}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ft, err := detectFromAPI(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ft != GitHub {
		t.Errorf("want GitHub, got %s", ft)
	}
}

// Mock forge for routing tests

type mockForge struct {
	repo      *Repository
	tags      []Tag
	lastOwner string
	lastRepo  string
}

func (m *mockForge) FetchRepository(_ context.Context, owner, repo string) (*Repository, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.repo, nil
}

func (m *mockForge) FetchTags(_ context.Context, owner, repo string) ([]Tag, error) {
	m.lastOwner = owner
	m.lastRepo = repo
	return m.tags, nil
}
