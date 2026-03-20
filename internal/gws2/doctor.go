package gws2

import "log/slog"

type DoctorRule int

const (
	DuplicateWorkspace DoctorRule = iota
	DuplicateProject
)

type DoctorRulesMap = map[DoctorRule]bool

// IsValid checks if a workspace is in a valid state without recursively checking children
func (w *Workspace) IsValid() bool {
	slog.Debug("Running doctor on workspace", "path", w.Path)
	rulesState := DoctorRulesMap{}

	rulesState[DuplicateWorkspace] = hasDuplicate(w.Children)
	rulesState[DuplicateProject] = hasDuplicate(w.Projects)

	slog.Debug("Doctor rules completed", "path", w.Path, "rules", rulesState)

	for _, state := range rulesState {
		if state {
			return false
		}
	}
	return true
}

func hasDuplicate[T Repository](repositories []T) bool {
	seen := make(map[string]struct{}, len(repositories))
	for _, r := range repositories {
		if _, exists := seen[r.GetPath()]; exists {
			return true
		}
		seen[r.GetPath()] = struct{}{}
	}
	return false
}
