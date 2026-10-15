package loader

import (
	"fmt"
	"strings"
)

type WorkspaceMode string

const (
	ModeNearest WorkspaceMode = "nearest"
	ModeTopmost WorkspaceMode = "topmost"
	ModeGlobal  WorkspaceMode = "global"
)

var WorkspaceModes = []WorkspaceMode{ModeNearest, ModeTopmost, ModeGlobal}

func ParseWorkspaceMode(raw string) (WorkspaceMode, error) {
	for _, mode := range WorkspaceModes {
		if strings.EqualFold(raw, string(mode)) {
			return mode, nil
		}
	}
	names := make([]string, len(WorkspaceModes))
	for i, mode := range WorkspaceModes {
		names[i] = string(mode)
	}
	return "", fmt.Errorf("invalid workspace mode %q: must be one of %s", raw, strings.Join(names, ", "))
}
