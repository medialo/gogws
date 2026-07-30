package git

import (
	"os"
	"path/filepath"

	"github.com/medialo/gogws/internal/gws"
)

func ToGitRemotes(remotes []gws.Remote) []Remote {
	result := make([]Remote, len(remotes))
	for i, r := range remotes {
		result[i] = Remote{Name: r.Name, URL: r.URL}
	}
	return result
}

//func ToGitRemotes2(remotes []gws2.Remote) []Remote {
//	result := make([]Remote, len(remotes))
//	for i, r := range remotes {
//		result[i] = Remote{Name: r.Name, URL: r.URL}
//	}
//	return result
//}

func IsGitFolder(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}
