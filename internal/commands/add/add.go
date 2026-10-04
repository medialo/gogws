package add

import (
	"github.com/medialo/gogws/internal/config"

	"github.com/spf13/cobra"
)

func NewCommand(getConfig func() *config.RunContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add repositories to gws file",
		Long:  `Add a project or workspace repositories to the gws configuration file.`,
	}

	cmd.AddCommand(newAddProjectCommand(getConfig))
	cmd.AddCommand(newAddWorkspaceCommand(getConfig))
	cmd.AddCommand(newAddCurrentCommand(getConfig))

	return cmd
}
