package standard

import (
	"errors"

	"github.com/medialo/gogws/internal/git"
)

const CurrentVersion = 1

type document struct {
	Version    int            `yaml:"version" json:"version" toml:"version"`
	Self       *selfEntry     `yaml:"self,omitempty" json:"self,omitempty" toml:"self,omitempty"`
	Workspaces []entry        `yaml:"workspaces,omitempty" json:"workspaces,omitempty" toml:"workspaces,omitempty"`
	Projects   []entry        `yaml:"projects,omitempty" json:"projects,omitempty" toml:"projects,omitempty"`
	Settings   map[string]any `yaml:"settings,omitempty" json:"settings,omitempty" toml:"settings,omitempty"`
}

type selfEntry struct {
	Remotes []remote `yaml:"remotes,omitempty" json:"remotes,omitempty" toml:"remotes,omitempty"`
}

type entry struct {
	Path    string   `yaml:"path,omitempty" json:"path,omitempty" toml:"path,omitempty"`
	Remotes []remote `yaml:"remotes,omitempty" json:"remotes,omitempty" toml:"remotes,omitempty"`
	Url     string   `yaml:"url,omitempty" json:"url,omitempty" toml:"url,omitempty"`
	Folder  string   `yaml:"folder,omitempty" json:"folder,omitempty" toml:"folder,omitempty"`
}

type remote struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	URL  string `yaml:"url" json:"url" toml:"url"`
}

func (d *document) isEmpty() bool {
	return d.Self == nil && len(d.Workspaces) == 0 && len(d.Projects) == 0 && len(d.Settings) == 0
}

func (e entry) checkWorkspace() error {
	if e.Folder != "" {
		if e.Path != "" || e.Url != "" || len(e.Remotes) > 0 {
			return errors.New("folder cannot be combined with path, url or remotes")
		}
		return nil
	}
	if e.Path == "" {
		return errors.New("path or folder is required")
	}
	return e.checkRemotes()
}

func (e entry) checkProject() error {
	if e.Folder != "" {
		return errors.New("folder is not allowed on a project")
	}
	if e.Path == "" {
		return errors.New("path is required")
	}
	if err := e.checkRemotes(); err != nil {
		return err
	}
	if e.Url == "" && len(e.Remotes) == 0 {
		return errors.New("url or remotes is required")
	}
	return nil
}

func (e entry) checkRemotes() error {
	if e.Url != "" && len(e.Remotes) > 0 {
		return errors.New("url cannot be combined with remotes")
	}
	return nil
}

func (e entry) location() string {
	if e.Folder != "" {
		return e.Folder
	}
	return e.Path
}

func (e entry) toRemotes() ([]*git.Remote, error) {
	if e.Url != "" {
		return []*git.Remote{{Name: defaultRemoteName(0), URL: e.Url}}, nil
	}
	return toRemotes(e.Remotes)
}
