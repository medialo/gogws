package doctor

import (
	"fmt"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/ui/cli"

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
	if cfg == nil {
		return fmt.Errorf("no workspace found (no .projects.gws file)")
	}

	ws, err := gws2.NewFromPath(cfg.WorkspaceRoot).RunDoctor(false).Load()
	if err != nil {
		return err
	}

	var results []gws2.WorkspaceDoctorResult
	if len(ids) == 0 {
		results, err = ws.RunAllDoctorChecksRecursive(autoFix)
	} else {
		results, err = ws.RunDoctorChecksRecursive(ids, autoFix)
	}
	if err != nil {
		return err
	}

	rendered := cli.NewRenderer()
	fmt.Println(rendered.RenderDoctorRun(ws, results))
	return nil
}
