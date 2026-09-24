// Package providers discovers a git provider organization/group's repos
// and subgroups via that provider's REST API, so gogws can auto-generate
// the .projects.gws/.workspaces.gws files that mirror its structure. Each
// provider lives in its own file (github.go, gitlab.go) implementing the
// small Provider interface; provider.go holds the shared types and the
// HTTP helper they both use.
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProviderProject is one repository discovered under a group/org.
type ProviderProject struct {
	Slug     string // filesystem-safe; becomes the project's RelativePath
	CloneURL string
}

// ProviderGroup is one organization/group discovered from a provider,
// with its repos and (for providers that support them) nested subgroups.
type ProviderGroup struct {
	Slug string // filesystem-safe; becomes the workspace's RelativePath
	// Ref is usable, combined with the provider's "<name>:" prefix, to
	// re-Discover this exact group later (e.g. "my-group/subgroup" for a
	// public GitLab subgroup, or a full URL for one on a self-hosted
	// instance so the host isn't lost). Only meaningful for entries
	// reachable as a Subgroups member — a subgroup gets written back as a
	// child workspace's remote so the next update pass can find it again.
	Ref       string
	Projects  []ProviderProject
	Subgroups []*ProviderGroup
}

// Provider resolves a "<name>:<ref>" workspace remote to its group's repos
// and subgroups.
type Provider interface {
	// Name identifies the provider, and is also its URL prefix (e.g.
	// "github", matching the "github:" remote prefix).
	Name() string
	// Match reports whether rawURL carries this provider's explicit
	// prefix ("github:" / "gitlab:"). Pure string check, no I/O.
	Match(rawURL string) bool
	// Discover resolves rawURL (still carrying this provider's "<name>:"
	// prefix — Discover strips it) to its group's repos and subgroups,
	// recursing into subgroups up to maxDepth.
	Discover(ctx context.Context, rawURL string, maxDepth int) (*ProviderGroup, error)
}

const (
	// MaxDiscoverDepth caps subgroup recursion during Discover. Shares
	// its default with gws2.DefaultMaxDepth (10) but is kept as its own
	// constant: it caps a different phase (API-side discovery, not
	// filesystem-side load recursion).
	MaxDiscoverDepth = 10
	// MaxReposPerDiscover defensively caps how many repos a single
	// Discover call will collect regardless of what a provider claims,
	// against a misbehaving or unexpectedly huge response.
	MaxReposPerDiscover = 500
)

var registry = []Provider{&GitHubProvider{}, &GitLabProvider{}}

// Find returns the first registered Provider whose Match(rawURL) is true,
// or nil if none match.
func Find(rawURL string) Provider {
	for _, p := range registry {
		if p.Match(rawURL) {
			return p
		}
	}
	return nil
}

// notFoundError marks an error as "the provider confirmed this org/group
// does not exist", so IsNotFoundError (and update's --prune) can treat it
// the same way git.IsNotFoundError treats a dead git remote.
type notFoundError struct {
	msg string
}

func (e *notFoundError) Error() string { return e.msg }

func newNotFoundError(format string, args ...any) error {
	return &notFoundError{msg: fmt.Sprintf(format, args...)}
}

// IsNotFoundError reports whether err indicates the provider confirmed the
// referenced organization/group does not exist, rather than a transient
// network or auth failure.
func IsNotFoundError(err error) bool {
	var nf *notFoundError
	return errors.As(err, &nf)
}

var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// getJSON performs an authenticated GET against url, JSON-decodes the
// response body into out (skipped if out is nil), and returns the next
// page URL (from a Link: rel="next" header — used by both GitHub's and
// GitLab's REST APIs) or "" once there are no more pages. Retries once on
// 429/503, honoring Retry-After, since Discover jobs run concurrently
// (one per engine.Job) and can burst against the same host.
func getJSON(ctx context.Context, url, token string, out any) (string, error) {
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			return "", newNotFoundError("not found: %s", url)
		}

		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) && attempt == 0 {
			wait := retryAfter(resp.Header)
			resp.Body.Close()
			select {
			case <-time.After(wait):
				continue
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return "", fmt.Errorf("unexpected status %d from %s: %s", resp.StatusCode, url, strings.TrimSpace(string(body)))
		}

		var decodeErr error
		if out != nil {
			decodeErr = json.NewDecoder(resp.Body).Decode(out)
		}
		next := ""
		if m := linkNextRe.FindStringSubmatch(resp.Header.Get("Link")); m != nil {
			next = m[1]
		}
		resp.Body.Close()
		if decodeErr != nil {
			return "", fmt.Errorf("failed to decode response from %s: %w", url, decodeErr)
		}
		return next, nil
	}
}

func retryAfter(h http.Header) time.Duration {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 2 * time.Second
}
