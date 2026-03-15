package git

import (
	"gogws/internal/gws"
	"gogws/internal/gws2"
)

func ToGitRemotes(remotes []gws.Remote) []Remote {
	result := make([]Remote, len(remotes))
	for i, r := range remotes {
		result[i] = Remote{Name: r.Name, URL: r.URL}
	}
	return result
}

func ToGitRemotes2(remotes []gws2.Remote) []Remote {
	result := make([]Remote, len(remotes))
	for i, r := range remotes {
		result[i] = Remote{Name: r.Name, URL: r.URL}
	}
	return result
}
