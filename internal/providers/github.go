package providers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/medialo/gogws/internal/config"
)

// GitHubProvider discovers a GitHub organization's (or user's) repos via
// the GitHub REST API. GitHub has no nested-organization concept, so
// Discover never populates Subgroups.
type GitHubProvider struct{}

func (p *GitHubProvider) Name() string { return "github" }

func (p *GitHubProvider) Match(rawURL string) bool {
	return strings.HasPrefix(rawURL, "github:")
}

type githubAccount struct {
	Type string `json:"type"` // "Organization" or "User"
}

type githubRepo struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
}

func (p *GitHubProvider) Discover(ctx context.Context, rawURL string, maxDepth int) (*ProviderGroup, error) {
	apiBase, owner, err := parseGitHubRef(strings.TrimPrefix(rawURL, "github:"))
	if err != nil {
		return nil, err
	}

	token := config.GetProviderToken("github")

	var account githubAccount
	if _, err := getJSON(ctx, apiBase+"/users/"+owner, token, &account); err != nil {
		return nil, err
	}

	reposURL := apiBase + "/orgs/" + owner + "/repos?per_page=100"
	if account.Type == "User" {
		reposURL = apiBase + "/users/" + owner + "/repos?per_page=100"
	}

	var projects []ProviderProject
	for reposURL != "" && len(projects) < MaxReposPerDiscover {
		var page []githubRepo
		next, err := getJSON(ctx, reposURL, token, &page)
		if err != nil {
			return nil, err
		}
		for _, r := range page {
			projects = append(projects, ProviderProject{Slug: r.Name, CloneURL: r.CloneURL})
			if len(projects) >= MaxReposPerDiscover {
				break
			}
		}
		reposURL = next
	}

	return &ProviderGroup{Slug: owner, Ref: githubRefFor(apiBase, owner), Projects: projects}, nil
}

// githubRefFor is the inverse of parseGitHubRef: given the API base used
// to reach an org/user and its slug, produces the string to store (after
// the "github:" prefix) that resolves back to the same host later. Mostly
// unused today (GitHub has no nested-org concept, so no subgroup ever
// needs to round-trip through this), kept for symmetry with GitLab's
// equivalent and in case that ever changes.
func githubRefFor(apiBase, owner string) string {
	if apiBase == "https://api.github.com" {
		return owner
	}
	return strings.TrimSuffix(apiBase, "/api/v3") + "/" + owner
}

// parseGitHubRef splits a "github:" remote's ref (prefix already
// stripped) into the REST API base to call and the org/user slug. A bare
// ref ("mon-org", "orgs/mon-org") resolves against the public
// api.github.com; a full URL resolves against its own host — github.com
// uses api.github.com, any other host is treated as a GitHub Enterprise
// Server instance, whose API lives at <host>/api/v3.
func parseGitHubRef(ref string) (apiBase, owner string, err error) {
	if !strings.Contains(ref, "://") {
		owner = strings.Trim(strings.TrimPrefix(ref, "orgs/"), "/")
		if owner == "" {
			return "", "", fmt.Errorf("empty github organization/user reference")
		}
		return "https://api.github.com", owner, nil
	}

	u, err := url.Parse(ref)
	if err != nil {
		return "", "", fmt.Errorf("invalid github url %q: %w", ref, err)
	}

	path := strings.Trim(u.Path, "/")
	path = strings.TrimPrefix(path, "orgs/")
	owner = strings.SplitN(path, "/", 2)[0]
	if owner == "" {
		return "", "", fmt.Errorf("invalid github organization url %q", ref)
	}

	if u.Host == "github.com" {
		return "https://api.github.com", owner, nil
	}
	return fmt.Sprintf("%s://%s/api/v3", u.Scheme, u.Host), owner, nil
}
