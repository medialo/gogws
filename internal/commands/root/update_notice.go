package root

import (
	"context"
	"os"
	"slices"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/interactive"
	"github.com/medialo/gogws/internal/ui/cli"
	"github.com/medialo/gogws/internal/updatecheck"
	"github.com/spf13/cobra"
)

const (
	updateNoticeMaxWait        = time.Second
	versionUpdateNoticeMaxWait = updatecheck.RequestTimeout
)

var (
	showVersion        bool
	appVersion         string
	updateCheck        *updatecheck.Check
	updateCheckStarted time.Time
	updateCheckMaxWait = updateNoticeMaxWait
	skipUpdateCheck    = []string{"completion", "__complete", "__completeNoDesc", "help"}
)

func SetVersion(v string) {
	appVersion = v
}

func VersionRequested() bool {
	return showVersion
}

func isVersionRequest(cmd *cobra.Command) bool {
	return showVersion && !cmd.HasParent()
}

func startUpdateCheck(cmd *cobra.Command, prefs *config.Preferences) {
	if !prefs.UpdateCheck.Value {
		return
	}

	force := isVersionRequest(cmd)
	if !force && (!interactive.Enabled() || slices.Contains(skipUpdateCheck, cmd.Name())) {
		return
	}

	stateDir, err := config.GetUserConfigDir()
	if err != nil {
		return
	}
	if force {
		updateCheckMaxWait = versionUpdateNoticeMaxWait
	}
	updateCheckStarted = time.Now()
	updateCheck = updatecheck.Start(context.Background(), updatecheck.Options{
		Current:  appVersion,
		StateDir: stateDir,
		Token:    config.GetProviderToken("github"),
		Force:    force,
	})
}

func PrintUpdateNotice() {
	notice := updateCheck.Notice(updateCheckStarted.Add(updateCheckMaxWait))
	if notice == "" {
		return
	}
	lipgloss.Fprintln(os.Stderr, "")
	lipgloss.Fprintln(os.Stderr, cli.NewRenderer().RenderWarning(notice))
}
