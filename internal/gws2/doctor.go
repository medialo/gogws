package gws2

import (
	"fmt"
	"log/slog"
	"strconv"
)

//go:generate enumer -type=DoctorCheckId
const (
	DuplicateWorkspace DoctorCheckId = iota
	DuplicateProject
)

type DoctorCheckId int

type DoctorCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	AutoFix     bool   `json:"auto_fix"` // AutoFix indicates whether the rule needs some interactivity to fix the issue
	test        func(w *Workspace) CheckResult
	Fix         func(w *Workspace) error
}

var DoctorRules = map[DoctorCheckId]DoctorCheck{
	DuplicateWorkspace: {
		Name:        "DuplicateWorkspace",
		Description: "Duplicate workspace",
		test:        func(w *Workspace) CheckResult { return hasDuplicate(w.Children) },
		Fix:         fixDuplicateWorkspaces,
		AutoFix:     true,
	},
	DuplicateProject: {
		Name:        "DuplicateProject",
		Description: "Duplicate project",
		test:        func(w *Workspace) CheckResult { return hasDuplicate(w.Projects) },
		Fix:         fixDuplicateProjects,
		AutoFix:     true,
	},
}

type CheckResult int

const (
	Passed CheckResult = iota
	Failed
	Fixed
)

type DoctorCheckResults map[DoctorCheckId]CheckResult

func (d *DoctorCheckResults) Total() int {
	return len(*d)
}

// Stats returns the number of passed, warning and failed checks
func (d *DoctorCheckResults) Stats() (int, int) {
	passed := 0
	failed := 0
	for _, r := range *d {
		if r == Passed {
			passed++
		} else {
			failed++
		}
	}
	return passed, failed
}

func (w *Workspace) RunAllDoctorChecks(autoFix bool) (DoctorCheckResults, error) {
	slog.Debug("Running all doctor checks", "ids", DoctorCheckIdStrings())
	return w.RunDoctorChecks(DoctorCheckIdStrings(), autoFix)
}

func (w *Workspace) RunDoctorChecks(checkIds []string, autoFix bool) (DoctorCheckResults, error) {
	result := DoctorCheckResults{}
	for _, stringId := range checkIds {
		ok, checkId, err := w.RunDoctorCheckId(stringId, autoFix)
		if err != nil {
			return nil, err
		}
		slog.Debug("Doctor check completed", "strId", stringId, "id", checkId, "ok", ok)
		result[checkId] = ok
	}
	return result, nil
}

func (w *Workspace) RunDoctorCheckId(id string, autoFix bool) (CheckResult, DoctorCheckId, error) {
	slog.Debug("Running doctor check", "id", id)
	var checkId *DoctorCheckId = nil

	// try to find check by number id
	idNumber, err := strconv.Atoi(id)
	if err == nil { // no error
		slog.Debug("Found doctor check by number id", "id", idNumber)
		checkId = ptr(DoctorCheckId(idNumber))
	}

	// try to find check by string id
	if checkId == nil || !(*checkId).IsADoctorCheckId() {
		id, err := DoctorCheckIdString(id)
		checkId = &id
		if err != nil {
			return Passed, *checkId, fmt.Errorf("invalid doctor check id: %w", err)
		}
	}

	check := DoctorRules[*checkId]
	checkResult := check.test(w)

	if checkResult == Failed && autoFix && check.Fix != nil {
		if err := check.Fix(w); err != nil {
			return Failed, *checkId, err
		}
		checkResult = Fixed
	}

	return checkResult, *checkId, nil
}

// WorkspaceDoctorResult pairs one workspace node with the outcome of the
// checks run on it, so a recursive run can report exactly which nested
// workspace has the issue.
type WorkspaceDoctorResult struct {
	Workspace *Workspace
	Results   DoctorCheckResults
}

// RunAllDoctorChecksRecursive runs every registered check on w and every
// descendant workspace already loaded in the tree.
func (w *Workspace) RunAllDoctorChecksRecursive(autoFix bool) ([]WorkspaceDoctorResult, error) {
	return w.RunDoctorChecksRecursive(DoctorCheckIdStrings(), autoFix)
}

// RunDoctorChecksRecursive runs checkIds on w and every descendant
// workspace already loaded in the tree, since a duplicate project or
// workspace can occur just as well inside a nested workspace as at the
// root.
func (w *Workspace) RunDoctorChecksRecursive(checkIds []string, autoFix bool) ([]WorkspaceDoctorResult, error) {
	nodes := append([]*Workspace{w}, w.FlattenWorkspaces()...)
	results := make([]WorkspaceDoctorResult, 0, len(nodes))
	for _, node := range nodes {
		r, err := node.RunDoctorChecks(checkIds, autoFix)
		if err != nil {
			return nil, err
		}
		results = append(results, WorkspaceDoctorResult{Workspace: node, Results: r})
	}
	return results, nil
}

// IsValid checks if a workspace is in a valid state without recursively checking children
func (w *Workspace) IsValid() bool {
	slog.Debug("Running doctor on workspace", "path", w.AbsolutePath)

	rulesState := DoctorCheckResults{}
	for id, rule := range DoctorRules {
		rulesState[id] = rule.test(w)
	}

	slog.Debug("Doctor rules completed", "path", w.AbsolutePath, "rules", rulesState)

	for _, state := range rulesState {
		if state == Failed {
			return false
		}
	}
	return true
}

func hasDuplicate[T Repository](repositories []T) CheckResult {
	seen := make(map[string]struct{}, len(repositories))
	for _, r := range repositories {
		key := pathKey(r.GetPath())
		if _, exists := seen[key]; exists {
			return Failed
		}
		seen[key] = struct{}{}
	}
	return Passed
}

// fixDuplicateWorkspaces drops every child workspace whose path repeats an
// earlier one, keeping the first occurrence (i.e. the earliest line in
// .workspaces.gws), then persists the deduplicated list.
//
// Deliberately does not call w.ReindexAll(): w here is whichever node the
// check ran on (root or a nested workspace, from RunDoctorChecksRecursive),
// and Index.Rebuild always rebuilds from the tree's actual root — calling
// it with a nested w would wipe the shared index down to just that
// subtree. The on-disk file is the source of truth for the next run, and
// nothing later in this process depends on the in-memory index.
func fixDuplicateWorkspaces(w *Workspace) error {
	seen := make(map[string]struct{}, len(w.Children))
	deduped := w.Children[:0]
	for _, c := range w.Children {
		key := pathKey(c.AbsolutePath)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, c)
	}
	w.Children = deduped
	return w.SaveWorkspace()
}

// fixDuplicateProjects drops every project whose path repeats an earlier
// one, keeping the first occurrence (i.e. the earliest line in
// .projects.gws), then persists the deduplicated list. See
// fixDuplicateWorkspaces for why this doesn't reindex.
func fixDuplicateProjects(w *Workspace) error {
	seen := make(map[string]struct{}, len(w.Projects))
	deduped := w.Projects[:0]
	for _, p := range w.Projects {
		key := pathKey(p.AbsolutePath)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, p)
	}
	w.Projects = deduped
	return w.SaveProjects()
}
