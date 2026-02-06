# forges

Go module for fetching normalized repository metadata from git forges. Supports GitHub, GitLab, Gitea/Forgejo, and Bitbucket Cloud.

```go
import "github.com/git-pkgs/forges"
```

## Usage

```go
client := forges.NewClient(
    forges.WithToken("github.com", os.Getenv("GITHUB_TOKEN")),
    forges.WithToken("gitlab.com", os.Getenv("GITLAB_TOKEN")),
)

repo, err := client.FetchRepository(ctx, "https://github.com/octocat/hello-world")
// repo.FullName == "octocat/hello-world"
// repo.License == "MIT"
// repo.StargazersCount == 12345

tags, err := client.FetchTags(ctx, "https://github.com/octocat/hello-world")
// tags[0].Name == "v1.0.0"
// tags[0].Commit == "abc123..."
```

Self-hosted instances can be registered with `WithGitea` or `WithGitLab`:

```go
client := forges.NewClient(
    forges.WithGitea("gitea.example.com", token),
    forges.WithGitLab("gitlab.internal.dev", token),
)
```

Or detected automatically:

```go
err := client.RegisterDomain(ctx, "git.example.com", token)
```

PURL support via the `github.com/git-pkgs/purl` module:

```go
p, _ := purl.Parse("pkg:npm/lodash?repository_url=https://github.com/lodash/lodash")
repo, err := client.FetchRepositoryFromPURL(ctx, p)
```

## Repository fields

FullName, Owner, Name, Description, Homepage, HTMLURL, Language, License (SPDX key), DefaultBranch, Fork, Archived, Private, MirrorURL, SourceName, Size, StargazersCount, ForksCount, OpenIssuesCount, SubscribersCount, HasIssues, PullRequestsEnabled, Topics, LogoURL, CreatedAt, UpdatedAt, PushedAt.
