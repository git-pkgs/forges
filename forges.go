package forges

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/git-pkgs/purl"
)

// ErrNotFound is returned when the requested repository does not exist.
var ErrNotFound = errors.New("repository not found")

// HTTPError represents a non-OK HTTP response from a forge API.
type HTTPError struct {
	StatusCode int
	URL        string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("forge: HTTP %d from %s", e.StatusCode, e.URL)
}

// Forge is the interface each forge backend implements.
type Forge interface {
	FetchRepository(ctx context.Context, owner, repo string) (*Repository, error)
	FetchTags(ctx context.Context, owner, repo string) ([]Tag, error)
}

// Client routes requests to the appropriate Forge based on the URL domain.
type Client struct {
	forges     map[string]Forge
	tokens     map[string]string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithToken sets the API token for the given domain.
func WithToken(domain, token string) Option {
	return func(c *Client) {
		c.tokens[domain] = token
	}
}

// WithHTTPClient overrides the default HTTP client used by forge backends.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithGitea registers a self-hosted Gitea or Forgejo instance.
func WithGitea(domain, token string) Option {
	return func(c *Client) {
		c.tokens[domain] = token
		c.forges[domain] = newGiteaForge("https://"+domain, token, c.httpClient)
	}
}

// WithGitLab registers a self-hosted GitLab instance.
func WithGitLab(domain, token string) Option {
	return func(c *Client) {
		c.tokens[domain] = token
		c.forges[domain] = newGitLabForge("https://"+domain, token, c.httpClient)
	}
}

// NewClient creates a Client with the default forge registrations and applies
// the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		forges: make(map[string]Forge),
		tokens: make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}

	// Register defaults. Tokens may have been set via WithToken before this runs.
	if _, ok := c.forges["github.com"]; !ok {
		c.forges["github.com"] = newGitHubForge(c.tokens["github.com"], c.httpClient)
	}
	if _, ok := c.forges["gitlab.com"]; !ok {
		c.forges["gitlab.com"] = newGitLabForge("https://gitlab.com", c.tokens["gitlab.com"], c.httpClient)
	}
	if _, ok := c.forges["codeberg.org"]; !ok {
		c.forges["codeberg.org"] = newGiteaForge("https://codeberg.org", c.tokens["codeberg.org"], c.httpClient)
	}
	if _, ok := c.forges["bitbucket.org"]; !ok {
		c.forges["bitbucket.org"] = newBitbucketForge(c.tokens["bitbucket.org"], c.httpClient)
	}
	return c
}

// RegisterDomain detects the forge type for a domain and registers it.
func (c *Client) RegisterDomain(ctx context.Context, domain, token string) error {
	ft, err := DetectForgeType(ctx, domain)
	if err != nil {
		return fmt.Errorf("detecting forge type for %s: %w", domain, err)
	}
	c.tokens[domain] = token
	baseURL := "https://" + domain
	switch ft {
	case GitHub:
		c.forges[domain] = newGitHubForgeWithBase(baseURL, token, c.httpClient)
	case GitLab:
		c.forges[domain] = newGitLabForge(baseURL, token, c.httpClient)
	case Gitea, Forgejo:
		c.forges[domain] = newGiteaForge(baseURL, token, c.httpClient)
	default:
		return fmt.Errorf("unsupported forge type %q for %s", ft, domain)
	}
	return nil
}

func (c *Client) forgeFor(domain string) (Forge, error) {
	f, ok := c.forges[domain]
	if !ok {
		return nil, fmt.Errorf("no forge registered for domain %q", domain)
	}
	return f, nil
}

// FetchRepository fetches normalized repository metadata from a URL string.
func (c *Client) FetchRepository(ctx context.Context, repoURL string) (*Repository, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.FetchRepository(ctx, owner, repo)
}

// FetchRepositoryFromPURL fetches repository metadata using a PURL's
// repository_url qualifier.
func (c *Client) FetchRepositoryFromPURL(ctx context.Context, p *purl.PURL) (*Repository, error) {
	repoURL := p.RepositoryURL()
	if repoURL == "" {
		return nil, fmt.Errorf("PURL has no repository_url qualifier")
	}
	return c.FetchRepository(ctx, repoURL)
}

// FetchTags fetches git tags from a URL string.
func (c *Client) FetchTags(ctx context.Context, repoURL string) ([]Tag, error) {
	domain, owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}
	f, err := c.forgeFor(domain)
	if err != nil {
		return nil, err
	}
	return f.FetchTags(ctx, owner, repo)
}

// FetchTagsFromPURL fetches git tags using a PURL's repository_url qualifier.
func (c *Client) FetchTagsFromPURL(ctx context.Context, p *purl.PURL) ([]Tag, error) {
	repoURL := p.RepositoryURL()
	if repoURL == "" {
		return nil, fmt.Errorf("PURL has no repository_url qualifier")
	}
	return c.FetchTags(ctx, repoURL)
}

// ParseRepoURL extracts the domain, owner, and repo from a repository URL.
// It handles https://, schemeless, and git@host:owner/repo SSH URLs, and
// strips .git suffixes and extra path segments.
func ParseRepoURL(rawURL string) (domain, owner, repo string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", "", fmt.Errorf("empty URL")
	}

	// Handle git@ SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@") {
		rawURL = strings.TrimPrefix(rawURL, "git@")
		colonIdx := strings.Index(rawURL, ":")
		if colonIdx < 0 {
			return "", "", "", fmt.Errorf("invalid SSH URL: missing colon")
		}
		domain = rawURL[:colonIdx]
		path := rawURL[colonIdx+1:]
		return splitOwnerRepo(domain, path)
	}

	// Add scheme if missing
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}
	domain = u.Hostname()
	return splitOwnerRepo(domain, u.Path)
}

func splitOwnerRepo(domain, path string) (string, string, string, error) {
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("URL path must contain owner/repo, got %q", path)
	}
	return domain, parts[0], parts[1], nil
}
