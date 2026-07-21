package doctor

import (
	"fmt"
	"gogws/internal/config"
	"gogws/internal/gws2"
	"gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

var (
	autoFix bool
)

func newDoctorRunCommand(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run checks on the current workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctorRun(getConfig, args)
		},
	}

	cmd.Flags().BoolVar(&autoFix, "autofix", false, "Run checks and automatically fix issues if possible")

	return cmd
}

func runDoctorRun(getConfig func() *config.Config, ids []string) error {
	cfg := getConfig()

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Load()
	if err != nil {
		return err
	}

	var result gws2.DoctorCheckResults
	if len(ids) == 0 {
		result, err = ws.RunAllDoctorChecks(autoFix)
	} else {
		result, err = ws.RunDoctorChecks(ids, autoFix)
	}
	if err != nil {
		return err
	}

	rendered := cli.NewRenderer()
	fmt.Println(rendered.RenderDoctorRun(ws, result))
	return nil
}
