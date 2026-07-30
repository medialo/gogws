package view

import (
	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

type GitRepositoryStatusView struct {
	GitStatus     *git.RepositoryStatus
	GwsRepository gws2.Repository
}
