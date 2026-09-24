package providers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/medialo/gogws/internal/gws2"
)

// StaleWorkspaces walks root's already-loaded tree and returns every
// workspace whose remote matches a registered Provider and is already
// materialized on disk, where either force is true or its generated .gws
// files are older than ttl. It never returns a workspace with no
// provider-matching remote — a plain git workspace is never re-probed,
// even once, past its first successful clone.
func StaleWorkspaces(root *gws2.Workspace, ttl time.Duration, force bool) []*gws2.Workspace {
	var stale []*gws2.Workspace
	for _, ws := range root.FlattenWorkspaces() {
		if !ws.FolderExist() {
			continue // not materialized yet — the normal clone/discover flow handles it
		}
		remote := ws.GetOriginRemote()
		if remote == nil || Find(remote.URL) == nil {
			continue
		}
		if force || fetchedAt(ws.GetPath()).Add(ttl).Before(time.Now()) {
			stale = append(stale, ws)
		}
	}
	return stale
}

// fetchedAt returns the most recent modification time of the .gws files
// under dir, or the zero time if neither exists (treated as infinitely
// stale, so it's always eligible for refresh).
func fetchedAt(dir string) time.Time {
	var latest time.Time
	for _, name := range []string{".gws/.projects.gws", ".gws/.workspaces.gws"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest
}
