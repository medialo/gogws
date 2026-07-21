package doctor

import (
	"fmt"
	"gogws/internal/gws2"
	"gogws/internal/ui/cli"
	"log/slog"
	"strconv"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
)

func newDoctorListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available checks",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cobra.NoArgs(cmd, args)
			if err != nil {
				return err
			}
			return runDoctorList()
		},
	}
	return cmd
}

func runDoctorList() error {
	renderer := cli.NewRenderer()
	slog.Debug("Listing available checks")
	t := table.New().
		Border(lipgloss.NormalBorder()).
		Headers("Id", "Name", "Description", "Auto-fix")

	for id, rule := range gws2.DoctorRules {
		autoFix := ""
		if rule.AutoFix {
			autoFix = "Yes"
		} else {
			autoFix = "No"
		}
		t.Row(strconv.Itoa(int(id)), rule.Name, rule.Description, autoFix)
	}
	fmt.Println(t.Render())
	fmt.Println(renderer.Theme().Subtle.Render("Use `gogws doctor run <id> or <name>` to run a specific check"))
	return nil
}
