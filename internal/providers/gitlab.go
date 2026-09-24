package providers

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/medialo/gogws/internal/config"
)

// GitLabProvider discovers a GitLab group's projects and subgroups via
// the GitLab REST API, recursing into subgroups up to maxDepth.
type GitLabProvider struct{}

func (p *GitLabProvider) Name() string { return "gitlab" }

func (p *GitLabProvider) Match(rawURL string) bool {
	return strings.HasPrefix(rawURL, "gitlab:")
}

type gitlabGroup struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

type gitlabProject struct {
	Path          string `json:"path"`
	HTTPURLToRepo string `json:"http_url_to_repo"`
}

type gitlabSubgroup struct {
	FullPath string `json:"full_path"`
}

func (p *GitLabProvider) Discover(ctx context.Context, rawURL string, maxDepth int) (*ProviderGroup, error) {
	apiBase, fullPath, err := parseGitLabRef(strings.TrimPrefix(rawURL, "gitlab:"))
	if err != nil {
		return nil, err
	}
	return discoverGitLabGroup(ctx, apiBase, fullPath, 0, maxDepth)
}

func discoverGitLabGroup(ctx context.Context, apiBase, fullPath string, depth, maxDepth int) (*ProviderGroup, error) {
	token := config.GetProviderToken("gitlab")

	var group gitlabGroup
	// The "gitlab:" prefix is an explicit declaration that this is meant
	// to be a group, so a 404 here is a genuine "not found" error, not an
	// ambiguous case to fall back on.
	if _, err := getJSON(ctx, apiBase+"/groups/"+url.PathEscape(fullPath), token, &group); err != nil {
		return nil, err
	}

	slug := fullPath
	if idx := strings.LastIndex(fullPath, "/"); idx != -1 {
		slug = fullPath[idx+1:]
	}
	result := &ProviderGroup{Slug: slug, Ref: gitlabRefFor(apiBase, fullPath)}

	projectsURL := fmt.Sprintf("%s/groups/%d/projects?per_page=100&include_subgroups=false", apiBase, group.ID)
	for projectsURL != "" && len(result.Projects) < MaxReposPerDiscover {
		var page []gitlabProject
		next, err := getJSON(ctx, projectsURL, token, &page)
		if err != nil {
			return nil, err
		}
		for _, pr := range page {
			result.Projects = append(result.Projects, ProviderProject{Slug: pr.Path, CloneURL: pr.HTTPURLToRepo})
			if len(result.Projects) >= MaxReposPerDiscover {
				break
			}
		}
		projectsURL = next
	}

	if depth >= maxDepth {
		return result, nil
	}

	subgroupsURL := fmt.Sprintf("%s/groups/%d/subgroups?per_page=100", apiBase, group.ID)
	for subgroupsURL != "" {
		var page []gitlabSubgroup
		next, err := getJSON(ctx, subgroupsURL, token, &page)
		if err != nil {
			return nil, err
		}
		for _, sg := range page {
			child, err := discoverGitLabGroup(ctx, apiBase, sg.FullPath, depth+1, maxDepth)
			if err != nil {
				return nil, err
			}
			result.Subgroups = append(result.Subgroups, child)
		}
		subgroupsURL = next
	}

	return result, nil
}

// parseGitLabRef splits a "gitlab:" remote's ref (prefix already
// stripped) into the REST API base to call and the group's full path
// (e.g. "my-group/subgroup"). A bare ref resolves against the public
// gitlab.com; a full URL resolves against its own host — self-managed
// GitLab instances use the same <host>/api/v4 shape as gitlab.com, so no
// special-casing is needed beyond substituting the host.
func parseGitLabRef(ref string) (apiBase, fullPath string, err error) {
	if !strings.Contains(ref, "://") {
		fullPath = strings.Trim(ref, "/")
		if fullPath == "" {
			return "", "", fmt.Errorf("empty gitlab group reference")
		}
		return "https://gitlab.com/api/v4", fullPath, nil
	}

	u, err := url.Parse(ref)
	if err != nil {
		return "", "", fmt.Errorf("invalid gitlab url %q: %w", ref, err)
	}

	fullPath = strings.Trim(u.Path, "/")
	if fullPath == "" {
		return "", "", fmt.Errorf("invalid gitlab group url %q", ref)
	}

	return fmt.Sprintf("%s://%s/api/v4", u.Scheme, u.Host), fullPath, nil
}

// gitlabRefFor is the inverse of parseGitLabRef: given the API base used
// to reach a group and its full path, it produces the string to store
// (after the "gitlab:" prefix) so a later Discover resolves back to the
// same host and group. The public gitlab.com default collapses to a bare
// path for a shorter, cleaner remote line; any other host round-trips as
// a full URL so it isn't lost.
func gitlabRefFor(apiBase, fullPath string) string {
	if apiBase == "https://gitlab.com/api/v4" {
		return fullPath
	}
	return strings.TrimSuffix(apiBase, "/api/v4") + "/" + fullPath
}
