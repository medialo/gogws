package doctor

import (
	"gogws/internal/config"

	"github.com/spf13/cobra"
)

func NewDoctorCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"doc", "aie", "bobo"},
		Short:   "Run a diagnostic check on the current workspace",
	}

	cmd.AddCommand(newDoctorListCommand())
	cmd.AddCommand(newDoctorRunCommand(getConfig))
	return cmd
}
