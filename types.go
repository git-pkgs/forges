package forges

import "time"

// ForgeType identifies which forge software a domain runs.
type ForgeType string

const (
	GitHub    ForgeType = "github"
	GitLab    ForgeType = "gitlab"
	Gitea     ForgeType = "gitea"
	Forgejo   ForgeType = "forgejo"
	Bitbucket ForgeType = "bitbucket"
	Unknown   ForgeType = "unknown"
)

// Repository holds normalized metadata about a source code repository,
// independent of which forge hosts it.
type Repository struct {
	FullName            string    `json:"full_name"`
	Owner               string    `json:"owner"`
	Name                string    `json:"name"`
	Description         string    `json:"description,omitempty"`
	Homepage            string    `json:"homepage,omitempty"`
	HTMLURL             string    `json:"html_url"`
	Language            string    `json:"language,omitempty"`
	License             string    `json:"license,omitempty"` // SPDX identifier
	DefaultBranch       string    `json:"default_branch,omitempty"`
	Fork                bool      `json:"fork"`
	Archived            bool      `json:"archived"`
	Private             bool      `json:"private"`
	MirrorURL           string    `json:"mirror_url,omitempty"`
	SourceName          string    `json:"source_name,omitempty"` // fork parent full name
	Size                int       `json:"size"`
	StargazersCount     int       `json:"stargazers_count"`
	ForksCount          int       `json:"forks_count"`
	OpenIssuesCount     int       `json:"open_issues_count"`
	SubscribersCount    int       `json:"subscribers_count"`
	HasIssues           bool      `json:"has_issues"`
	PullRequestsEnabled bool      `json:"pull_requests_enabled"`
	Topics              []string  `json:"topics,omitempty"`
	LogoURL             string    `json:"logo_url,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	PushedAt            time.Time `json:"pushed_at,omitzero"`
}

// ArchivedFilter controls how archived repositories are handled in list operations.
type ArchivedFilter int

const (
	ArchivedInclude ArchivedFilter = iota
	ArchivedExclude
	ArchivedOnly
)

// ForkFilter controls how forked repositories are handled in list operations.
type ForkFilter int

const (
	ForkInclude ForkFilter = iota
	ForkExclude
	ForkOnly
)

// ListOptions configures a ListRepositories call.
type ListOptions struct {
	Archived ArchivedFilter
	Forks    ForkFilter
	PerPage  int
}

// Tag represents a git tag.
type Tag struct {
	Name   string `json:"name"`
	Commit string `json:"commit"` // SHA
}
