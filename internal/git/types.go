package git

import "path/filepath"

type BranchStatus struct {
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
	Upstream  string `json:"upstream,omitempty"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
}

type RepositoryStatus struct {
	Path        string         `json:"path"`
	Exists      bool           `json:"exists"`
	Clean       bool           `json:"clean"`
	Branch      string         `json:"branch"`
	Branches    []BranchStatus `json:"branches,omitempty"`
	Ahead       int            `json:"ahead"`
	Behind      int            `json:"behind"`
	Uncommitted int            `json:"uncommitted"`
	Untracked   int            `json:"untracked"`
	HasRemote   bool           `json:"has_remote"`
	Error       error          `json:"-"`
	Oid         string         `json:"oid"`
}

func (r *RepositoryStatus) Name() string {
	return filepath.Base(r.Path)
}
