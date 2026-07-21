package gws2

import (
	"fmt"
	"log/slog"
	"strconv"
)

//go:generate enumer -type=DoctorCheckId
const (
	WorkspaceRootMissing DoctorCheckId = iota
	DuplicateWorkspace
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
	WorkspaceRootMissing: {
		Name:        "WorkspaceRootMissing",
		Description: "Workspace root git url is missing in workspace.gws.",
		test:        func(w *Workspace) CheckResult { return Fixed },
		Fix: func(w *Workspace) error {
			w.AddSelfWorkspace()
			err := w.SaveAll()
			if err != nil {
				return err
			}
			return nil
		},
		AutoFix: true,
	},
	DuplicateWorkspace: {
		Name:        "DuplicateWorkspace",
		Description: "Duplicate workspace",
		test:        func(w *Workspace) CheckResult { return hasDuplicate(w.Children) },
		Fix:         nil,
		AutoFix:     false,
	},
	DuplicateProject: {
		Name:        "DuplicateProject",
		Description: "Duplicate project",
		test:        func(w *Workspace) CheckResult { return hasDuplicate(w.Projects) },
		Fix:         nil,
		AutoFix:     false,
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
		err := check.Fix(w)
		if err != nil {
			return Passed, *checkId, err
		}
	}

	return checkResult, *checkId, nil
}

// IsValid checks if a workspace is in a valid state without recursively checking children
func (w *Workspace) IsValid() bool {
	slog.Debug("Running doctor on workspace", "path", w.Path)

	rulesState := DoctorCheckResults{}
	for id, rule := range DoctorRules {
		rulesState[id] = rule.test(w)
	}

	slog.Debug("Doctor rules completed", "path", w.Path, "rules", rulesState)

	for _, state := range rulesState {
		if state == Failed {
			return false
		}
	}
	return true
}

func workspaceRootMissing(workspaces []*Workspace) bool {
	return false
}

func hasDuplicate[T Repository](repositories []T) CheckResult {
	seen := make(map[string]struct{}, len(repositories))
	for _, r := range repositories {
		if _, exists := seen[r.GetPath()]; exists {
			return Failed
		}
		seen[r.GetPath()] = struct{}{}
	}
	return Passed
}
