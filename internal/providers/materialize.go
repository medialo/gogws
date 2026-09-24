package providers

import (
	"os"

	"github.com/medialo/gogws/internal/git"
	"github.com/medialo/gogws/internal/gws2"
)

// Materialize recursively builds Project/Workspace entries for group
// under parent, using gws2's existing exported factories and Add/Save
// methods — no changes needed to gws2 itself. Subgroups are added as
// child workspaces with a remote of "<provider>:<ref>" so they round-trip
// through parseWorkspaceLine with zero special-casing and get picked up
// by the ordinary MissingWorkspaces()/cloneWorkspaces flow on the next
// update pass, exactly like any other workspace line.
func Materialize(parent *gws2.Workspace, provider Provider, group *ProviderGroup) error {
	root := parent.GetPath()

	// Create the directory unconditionally, even for a group with no
	// projects or subgroups: otherwise an empty provider group would
	// never exist on disk, so MissingWorkspaces() would keep seeing it as
	// missing and re-discover it (a wasted API call) on every update.
	if err := os.MkdirAll(root, 0755); err != nil {
		return err
	}
	parent.FolderExists = true

	for _, proj := range group.Projects {
		project := gws2.NewProject(root, proj.Slug, []*git.Remote{{Name: "origin", URL: proj.CloneURL}})
		parent.AddProject(project)
	}

	for _, sub := range group.Subgroups {
		remote := &git.Remote{Name: "origin", URL: provider.Name() + ":" + sub.Ref}
		parent.AddWorkspace(gws2.NewChildWorkspace(root, sub.Slug, remote))
	}

	if len(group.Projects) > 0 {
		if err := parent.SaveProjects(); err != nil {
			return err
		}
	}
	if len(group.Subgroups) > 0 {
		if err := parent.SaveWorkspace(); err != nil {
			return err
		}
	}

	return nil
}
