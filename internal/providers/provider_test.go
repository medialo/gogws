package providers

import "testing"

func TestGitHubProvider_Match(t *testing.T) {
	p := &GitHubProvider{}
	cases := map[string]bool{
		"github:mon-org":                         true,
		"github:orgs/mon-org":                    true,
		"github:https://github.com/orgs/mon-org": true,
		"gitlab:mon-groupe":                      false,
		"https://github.com/user/repo.git":       false,
		"":                                       false,
	}
	for url, want := range cases {
		if got := p.Match(url); got != want {
			t.Errorf("Match(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestGitLabProvider_Match(t *testing.T) {
	p := &GitLabProvider{}
	cases := map[string]bool{
		"gitlab:mon-groupe":                    true,
		"gitlab:https://gitlab.com/mon-groupe": true,
		"github:mon-org":                       false,
		"https://gitlab.com/user/repo.git":     false,
		"gitlab-graphql:mon-groupe":            false,
		"":                                     false,
	}
	for url, want := range cases {
		if got := p.Match(url); got != want {
			t.Errorf("Match(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestGitLabGraphQLProvider_Match(t *testing.T) {
	p := &GitLabGraphQLProvider{}
	cases := map[string]bool{
		"gitlab-graphql:mon-groupe":                    true,
		"gitlab-graphql:https://gitlab.com/mon-groupe": true,
		"gitlab:mon-groupe":                            false,
		"github:mon-org":                               false,
		"":                                             false,
	}
	for url, want := range cases {
		if got := p.Match(url); got != want {
			t.Errorf("Match(%q) = %v, want %v", url, got, want)
		}
	}
}

func TestFind_GitLabPrefixesDoNotCollide(t *testing.T) {
	if got := Find("gitlab:mon-groupe"); got == nil || got.Name() != "gitlab" {
		t.Errorf("Find(gitlab:...) = %v, want the gitlab provider", got)
	}
	if got := Find("gitlab-graphql:mon-groupe"); got == nil || got.Name() != "gitlab-graphql" {
		t.Errorf("Find(gitlab-graphql:...) = %v, want the gitlab-graphql provider", got)
	}
}

func TestLastPathSegmentAndParentPath(t *testing.T) {
	cases := []struct {
		fullPath   string
		wantSlug   string
		wantParent string
	}{
		{"mon-groupe", "mon-groupe", ""},
		{"mon-groupe/sous-groupe", "sous-groupe", "mon-groupe"},
		{"mon-groupe/sous-groupe/projet", "projet", "mon-groupe/sous-groupe"},
	}
	for _, c := range cases {
		if got := lastPathSegment(c.fullPath); got != c.wantSlug {
			t.Errorf("lastPathSegment(%q) = %q, want %q", c.fullPath, got, c.wantSlug)
		}
		if got := parentPath(c.fullPath); got != c.wantParent {
			t.Errorf("parentPath(%q) = %q, want %q", c.fullPath, got, c.wantParent)
		}
	}
}

func TestParseGitHubRef(t *testing.T) {
	cases := []struct {
		ref         string
		wantAPIBase string
		wantOwner   string
		wantErr     bool
	}{
		{"mon-org", "https://api.github.com", "mon-org", false},
		{"orgs/mon-org", "https://api.github.com", "mon-org", false},
		{"https://github.com/orgs/mon-org", "https://api.github.com", "mon-org", false},
		{"https://github.com/mon-org", "https://api.github.com", "mon-org", false},
		{"https://github.mycompany.com/mon-org", "https://github.mycompany.com/api/v3", "mon-org", false},
		{"", "", "", true},
	}
	for _, c := range cases {
		apiBase, owner, err := parseGitHubRef(c.ref)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseGitHubRef(%q): expected error, got none", c.ref)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseGitHubRef(%q): unexpected error: %v", c.ref, err)
		}
		if apiBase != c.wantAPIBase || owner != c.wantOwner {
			t.Errorf("parseGitHubRef(%q) = (%q, %q), want (%q, %q)", c.ref, apiBase, owner, c.wantAPIBase, c.wantOwner)
		}
	}
}

func TestParseGitLabRef(t *testing.T) {
	cases := []struct {
		ref         string
		wantAPIBase string
		wantPath    string
		wantErr     bool
	}{
		{"mon-groupe", "https://gitlab.com/api/v4", "mon-groupe", false},
		{"mon-groupe/sous-groupe", "https://gitlab.com/api/v4", "mon-groupe/sous-groupe", false},
		{"https://gitlab.com/mon-groupe", "https://gitlab.com/api/v4", "mon-groupe", false},
		{"https://gitlab.mycompany.com/mon-groupe/sous-groupe", "https://gitlab.mycompany.com/api/v4", "mon-groupe/sous-groupe", false},
		{"", "", "", true},
	}
	for _, c := range cases {
		apiBase, fullPath, err := parseGitLabRef(c.ref)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseGitLabRef(%q): expected error, got none", c.ref)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseGitLabRef(%q): unexpected error: %v", c.ref, err)
		}
		if apiBase != c.wantAPIBase || fullPath != c.wantPath {
			t.Errorf("parseGitLabRef(%q) = (%q, %q), want (%q, %q)", c.ref, apiBase, fullPath, c.wantAPIBase, c.wantPath)
		}
	}
}

func TestGitLabRefFor_RoundTrip(t *testing.T) {
	cases := []struct {
		apiBase  string
		fullPath string
		want     string
	}{
		{"https://gitlab.com/api/v4", "mon-groupe", "mon-groupe"},
		{"https://gitlab.com/api/v4", "mon-groupe/sous-groupe", "mon-groupe/sous-groupe"},
		{"https://gitlab.mycompany.com/api/v4", "mon-groupe", "https://gitlab.mycompany.com/mon-groupe"},
	}
	for _, c := range cases {
		got := gitlabRefFor(c.apiBase, c.fullPath)
		if got != c.want {
			t.Errorf("gitlabRefFor(%q, %q) = %q, want %q", c.apiBase, c.fullPath, got, c.want)
		}
		// Round trip: what we produce must parse back to the same base+path.
		apiBase, fullPath, err := parseGitLabRef(got)
		if err != nil {
			t.Fatalf("parseGitLabRef(%q) round-trip: unexpected error: %v", got, err)
		}
		if apiBase != c.apiBase || fullPath != c.fullPath {
			t.Errorf("round-trip parseGitLabRef(%q) = (%q, %q), want (%q, %q)", got, apiBase, fullPath, c.apiBase, c.fullPath)
		}
	}
}

func TestIsNotFoundError(t *testing.T) {
	if IsNotFoundError(nil) {
		t.Error("IsNotFoundError(nil) = true, want false")
	}
	if !IsNotFoundError(newNotFoundError("not found: %s", "x")) {
		t.Error("IsNotFoundError(notFoundError) = false, want true")
	}
	if IsNotFoundError(errPlain("boom")) {
		t.Error("IsNotFoundError(plain error) = true, want false")
	}
}

type errPlain string

func (e errPlain) Error() string { return string(e) }
