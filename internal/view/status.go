package view

import (
	"gogws/internal/git"
	"gogws/internal/gws2"
)

type GitRepositoryStatusView struct {
	GitStatus     *git.RepositoryStatus
	GwsRepository gws2.Repository
}
