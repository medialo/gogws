package doctor

import (
	"gogws/internal/config"

	"github.com/spf13/cobra"
)

var (
	runFix bool
)

func NewDoctorCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "doctor",
		Aliases: []string{"doc", "aie", "bobo"},
		Short:   "Run a diagnostic check on the current workspace",
		Long:    ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cobra.NoArgs(cmd, args)
			if err != nil {
				return err
			}
			return runDoctor()
		},
	}

	cmd.Flags().BoolVar(&runFix, "fix", false, "After running the doctor, start fixing issues.")
	return cmd
}

func runDoctor() error {

	return nil
}
