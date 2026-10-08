package standard

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
	Path    string   `yaml:"path" json:"path" toml:"path"`
	Remotes []remote `yaml:"remotes,omitempty" json:"remotes,omitempty" toml:"remotes,omitempty"`
}

type remote struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty" toml:"name,omitempty"`
	URL  string `yaml:"url" json:"url" toml:"url"`
}

func (d *document) isEmpty() bool {
	return d.Self == nil && len(d.Workspaces) == 0 && len(d.Projects) == 0 && len(d.Settings) == 0
}
