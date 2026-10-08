package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGitLabDiscover_ParallelSubgroupsKeepOrder(t *testing.T) {
	tree := map[string][]string{
		"root":        {"root/a", "root/b", "root/c"},
		"root/a":      {"root/a/deep"},
		"root/b":      nil,
		"root/c":      nil,
		"root/a/deep": nil,
	}
	ids := map[string]int{}
	paths := map[int]string{}
	id := 0
	for path := range tree {
		id++
		ids[path] = id
		paths[id] = path
	}

	var inFlight, maxInFlight atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			seen := maxInFlight.Load()
			if current <= seen || maxInFlight.CompareAndSwap(seen, current) {
				break
			}
		}

		rest := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/groups/")
		switch {
		case strings.HasSuffix(rest, "/projects"):
			var groupID int
			fmt.Sscanf(rest, "%d/projects", &groupID)
			slug := paths[groupID][strings.LastIndex(paths[groupID], "/")+1:]
			json.NewEncoder(w).Encode([]gitlabProject{{Path: slug + "-repo", HTTPURLToRepo: "https://example.com/" + paths[groupID] + ".git"}})
		case strings.HasSuffix(rest, "/subgroups"):
			var groupID int
			fmt.Sscanf(rest, "%d/subgroups", &groupID)
			var subgroups []gitlabSubgroup
			for _, child := range tree[paths[groupID]] {
				subgroups = append(subgroups, gitlabSubgroup{FullPath: child})
			}
			json.NewEncoder(w).Encode(subgroups)
		default:
			fullPath, _ := url.PathUnescape(rest)
			groupID, ok := ids[fullPath]
			if !ok {
				http.NotFound(w, r)
				return
			}
			json.NewEncoder(w).Encode(gitlabGroup{ID: groupID, Path: fullPath})
		}
	}))
	defer server.Close()

	group, err := (&GitLabProvider{}).Discover(context.Background(), "gitlab:"+server.URL+"/root", 5)
	if err != nil {
		t.Fatal(err)
	}

	var got []string
	var walk func(g *ProviderGroup, prefix string)
	walk = func(g *ProviderGroup, prefix string) {
		got = append(got, prefix+g.Slug)
		for _, sub := range g.Subgroups {
			walk(sub, prefix+"  ")
		}
	}
	walk(group, "")

	want := []string{"root", "  a", "    deep", "  b", "  c"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("tree = %q, want %q", got, want)
	}
	if len(group.Projects) != 1 || group.Projects[0].Slug != "root-repo" {
		t.Fatalf("root projects = %+v", group.Projects)
	}
	if maxInFlight.Load() > gitlabParallel {
		t.Fatalf("max concurrent requests = %d, want <= %d", maxInFlight.Load(), gitlabParallel)
	}
}

func TestGitLabDiscover_SubgroupErrorFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v4/groups/")
		switch {
		case strings.HasSuffix(rest, "/projects"):
			json.NewEncoder(w).Encode([]gitlabProject{})
		case strings.HasSuffix(rest, "/subgroups"):
			json.NewEncoder(w).Encode([]gitlabSubgroup{{FullPath: "root/missing"}, {FullPath: "root/also-missing"}})
		case rest == "root":
			json.NewEncoder(w).Encode(gitlabGroup{ID: 1, Path: "root"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if _, err := (&GitLabProvider{}).Discover(context.Background(), "gitlab:"+server.URL+"/root", 5); !IsNotFoundError(err) {
		t.Fatalf("err = %v, want not found", err)
	}
}
