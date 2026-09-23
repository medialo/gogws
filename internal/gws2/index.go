package gws2

import (
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

func pathKey(path string) string {
	clean := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(clean)
	}
	return clean
}

type Index struct {
	mu     sync.RWMutex
	byPath map[string]Repository
}

func NewIndex() *Index {
	return &Index{byPath: make(map[string]Repository)}
}

func (idx *Index) Get(path string) (Repository, bool) {
	if idx == nil {
		return nil, false
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	repo, ok := idx.byPath[pathKey(path)]
	return repo, ok
}

func (idx *Index) Has(path string) bool {
	_, ok := idx.Get(path)
	return ok
}

func (idx *Index) put(repo Repository) {
	if idx == nil || repo == nil {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.byPath[pathKey(repo.GetPath())] = repo
}

// SearchByName returns every Repository (project or workspace) whose name
// contains query (case-insensitive), ordered by name. Filter the result by
// GetType() if only projects (or only workspaces) are wanted.
func (idx *Index) SearchByName(query string) []Repository {
	results := make([]Repository, 0)
	if idx == nil {
		return results
	}

	query = strings.ToLower(query)
	idx.mu.RLock()
	for _, repo := range idx.byPath {
		if strings.Contains(strings.ToLower(repo.GetName()), query) {
			results = append(results, repo)
		}
	}
	idx.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool { return results[i].GetName() < results[j].GetName() })
	return results
}

// SearchByPath returns every Repository (project or workspace) whose path
// contains query (case-insensitive), ordered by path.
func (idx *Index) SearchByPath(query string) []Repository {
	results := make([]Repository, 0)
	if idx == nil {
		return results
	}

	query = strings.ToLower(query)
	idx.mu.RLock()
	for _, repo := range idx.byPath {
		if strings.Contains(strings.ToLower(repo.GetPath()), query) {
			results = append(results, repo)
		}
	}
	idx.mu.RUnlock()

	sort.Slice(results, func(i, j int) bool { return results[i].GetPath() < results[j].GetPath() })
	return results
}

// Rebuild clears the index and re-registers every workspace and project
// reachable from root. Call it after mutating a tree in a way that bypasses
// AddProject/AddWorkspace (e.g. appending to Projects/Children directly), so
// the index reflects the current state again.
func (idx *Index) Rebuild(root *Workspace) {
	if idx == nil || root == nil {
		return
	}
	idx.mu.Lock()
	idx.byPath = make(map[string]Repository)
	idx.mu.Unlock()
	idx.indexTree(root)
}

func (idx *Index) indexTree(w *Workspace) {
	idx.put(w)
	for _, p := range w.Projects {
		idx.put(p)
	}
	for _, c := range w.Children {
		idx.indexTree(c)
	}
}
