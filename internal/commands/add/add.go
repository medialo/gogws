package add

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add repositories to gws file",
		Long:  `Add a project or workspace repositories to the gws configuration file.`,
	}

	cmd.AddCommand(newAddProjectCommand())

	return cmd
}
