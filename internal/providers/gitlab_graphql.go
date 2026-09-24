package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/medialo/gogws/internal/config"
)

// GitLabGraphQLProvider discovers a GitLab group's projects and subgroups
// via the GitLab GraphQL API. Unlike GitLabProvider (REST), it fetches the
// entire descendant-group tree and every nested project in a couple of
// paginated queries instead of one REST round trip per subgroup — a real
// win on groups with many nested subgroups, at the cost of a separate,
// cursor-based pagination model that can't share getJSON with the REST
// providers. Kept as its own provider (rather than replacing the REST one)
// so both are available side by side.
type GitLabGraphQLProvider struct{}

func (p *GitLabGraphQLProvider) Name() string { return "gitlab-graphql" }

func (p *GitLabGraphQLProvider) Match(rawURL string) bool {
	return strings.HasPrefix(rawURL, "gitlab-graphql:")
}

const gitlabGroupQuery = `query($fullPath: ID!, $groupsAfter: String, $projectsAfter: String) {
  group(fullPath: $fullPath) {
    id
    fullPath
    descendantGroups(after: $groupsAfter, first: 100) {
      nodes { fullPath }
      pageInfo { hasNextPage endCursor }
    }
    projects(includeSubgroups: true, after: $projectsAfter, first: 100) {
      nodes { fullPath httpUrlToRepo }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

type gqlPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type gqlGroupResponse struct {
	Data struct {
		Group *struct {
			FullPath         string `json:"fullPath"`
			DescendantGroups struct {
				Nodes []struct {
					FullPath string `json:"fullPath"`
				} `json:"nodes"`
				PageInfo gqlPageInfo `json:"pageInfo"`
			} `json:"descendantGroups"`
			Projects struct {
				Nodes []struct {
					FullPath      string `json:"fullPath"`
					HTTPURLToRepo string `json:"httpUrlToRepo"`
				} `json:"nodes"`
				PageInfo gqlPageInfo `json:"pageInfo"`
			} `json:"projects"`
		} `json:"group"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (p *GitLabGraphQLProvider) Discover(ctx context.Context, rawURL string, maxDepth int) (*ProviderGroup, error) {
	// Reuses gitlab.go's REST-oriented parseGitLabRef/gitlabRefFor: the
	// "<host>/api/v4" base and bare-vs-full-URL ref rules are identical for
	// GraphQL, only the endpoint path (/api/graphql instead of /api/v4/...)
	// and the query shape differ.
	apiBase, rootFullPath, err := parseGitLabRef(strings.TrimPrefix(rawURL, "gitlab-graphql:"))
	if err != nil {
		return nil, err
	}
	graphqlURL := strings.TrimSuffix(apiBase, "/api/v4") + "/api/graphql"
	token := config.GetProviderToken("gitlab")

	var descendantPaths []string
	type projRef struct {
		fullPath, cloneURL string
	}
	var projects []projRef

	var groupsAfter, projectsAfter string
	groupsDone, projectsDone := false, false

	for !groupsDone || !projectsDone {
		variables := map[string]any{
			"fullPath":      rootFullPath,
			"groupsAfter":   cursorOrNil(groupsAfter),
			"projectsAfter": cursorOrNil(projectsAfter),
		}

		var resp gqlGroupResponse
		if err := postGraphQL(ctx, graphqlURL, token, gitlabGroupQuery, variables, &resp); err != nil {
			return nil, err
		}
		if resp.Data.Group == nil {
			if len(resp.Errors) > 0 {
				return nil, newNotFoundError("not found: %s (%s)", rootFullPath, resp.Errors[0].Message)
			}
			return nil, newNotFoundError("not found: %s", rootFullPath)
		}
		group := resp.Data.Group

		if !groupsDone {
			for _, n := range group.DescendantGroups.Nodes {
				descendantPaths = append(descendantPaths, n.FullPath)
			}
			if group.DescendantGroups.PageInfo.HasNextPage {
				groupsAfter = group.DescendantGroups.PageInfo.EndCursor
			} else {
				groupsDone = true
			}
		}

		if !projectsDone {
			for _, n := range group.Projects.Nodes {
				projects = append(projects, projRef{n.FullPath, n.HTTPURLToRepo})
				if len(projects) >= MaxReposPerDiscover {
					projectsDone = true
					break
				}
			}
			if !projectsDone {
				if group.Projects.PageInfo.HasNextPage {
					projectsAfter = group.Projects.PageInfo.EndCursor
				} else {
					projectsDone = true
				}
			}
		}
	}

	root := &ProviderGroup{Slug: lastPathSegment(rootFullPath), Ref: gitlabRefFor(apiBase, rootFullPath)}
	byPath := map[string]*ProviderGroup{rootFullPath: root}
	rootDepth := strings.Count(rootFullPath, "/")

	sort.Slice(descendantPaths, func(i, j int) bool {
		return strings.Count(descendantPaths[i], "/") < strings.Count(descendantPaths[j], "/")
	})

	for _, fullPath := range descendantPaths {
		if strings.Count(fullPath, "/")-rootDepth > maxDepth {
			continue
		}
		parent, ok := byPath[parentPath(fullPath)]
		if !ok {
			// Parent itself was dropped for exceeding maxDepth — this
			// descendant is deeper still, so drop it too.
			continue
		}
		group := &ProviderGroup{Slug: lastPathSegment(fullPath), Ref: gitlabRefFor(apiBase, fullPath)}
		parent.Subgroups = append(parent.Subgroups, group)
		byPath[fullPath] = group
	}

	for _, pr := range projects {
		parent, ok := byPath[parentPath(pr.fullPath)]
		if !ok {
			continue
		}
		parent.Projects = append(parent.Projects, ProviderProject{Slug: lastPathSegment(pr.fullPath), CloneURL: pr.cloneURL})
	}

	return root, nil
}

func cursorOrNil(cursor string) any {
	if cursor == "" {
		return nil
	}
	return cursor
}

func lastPathSegment(fullPath string) string {
	if idx := strings.LastIndex(fullPath, "/"); idx != -1 {
		return fullPath[idx+1:]
	}
	return fullPath
}

func parentPath(fullPath string) string {
	idx := strings.LastIndex(fullPath, "/")
	if idx == -1 {
		return ""
	}
	return fullPath[:idx]
}

// postGraphQL POSTs a GraphQL query+variables to url and JSON-decodes the
// response envelope into out. Retries once on 429/503, honoring
// Retry-After, matching getJSON's REST retry behavior — GraphQL Discover
// jobs run concurrently too.
func postGraphQL(ctx context.Context, url, token, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return err
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt == 0 {
			wait := retryAfter(resp.Header)
			resp.Body.Close()
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, url, strings.TrimSpace(string(respBody)))
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(out)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("failed to decode response from %s: %w", url, decodeErr)
		}
		return nil
	}
}
