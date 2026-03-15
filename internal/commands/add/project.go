package add

import (
	"context"
	"fmt"
	"gogws/internal/git"
	"gogws/internal/gws"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

var (
	autoClone bool
)

func newAddProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [git url] [foldername]",
		Short: "Add project or workpace to the current .gws",
		Long:  `Add a project or workspace to the current .projects.gws or .workspaces.gws file.`,
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(args)
		},
	}

	cmd.Flags().BoolVar(&autoClone, "auto-clone", false, "Automatically clone the repository after adding it to the workspace")
	return cmd
}

func runAdd(args []string) error {
	ws, err := gws.FindRoot()

	if err != nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	var gitProjectUrl, projectFolderName string

	if len(args) == 2 {
		gitProjectUrl = args[0]
		projectFolderName = args[1]
	} else {
		huh.NewInput().
			Title("Git url of project to add").
			Value(&gitProjectUrl).
			Run()

		huh.NewInput().
			Title("Folder name for the project").
			Value(&projectFolderName).
			Run()
	}

	projectToAdd := &gws.Project{
		Path: projectFolderName,
		Remotes: []gws.Remote{{
			Name: "origin",
			URL:  gitProjectUrl,
		}},
	}

	err = gws.AddProject(ws.Root, projectToAdd)

	if err != nil {
		return fmt.Errorf("failed to add project: %w", err)
	}

	if autoClone {
		remotes := git.ToGitRemotes(projectToAdd.Remotes)
		err := git.CloneWorkspace(context.Background(), ws.Root, projectToAdd.Path, remotes, nil)

		if err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	return nil
}
