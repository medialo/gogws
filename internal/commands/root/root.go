package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/gws2"
	"github.com/medialo/gogws/internal/hooks"
	"github.com/medialo/gogws/internal/interactive"
	"github.com/medialo/gogws/internal/log"
	"github.com/medialo/gogws/internal/theme"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	onlyChanges bool
	verbosity   int
	trustHooks    string
	workingDir    string
	noInteractive bool
)

var rootCmd = &cobra.Command{
	Use:   "gogws",
	Short: "Git Workspace Manager - Manage multiple git repositories with ease",
	Long: `gogws is a modern Git workspace management tool written in Go.
It helps you manage multiple git repositories in a workspace with
interactive TUI and beautiful CLI output.

Compatible with gws project files (.projects.gws)`,
	PersistentPreRunE: persistentPreRun,
}

func persistentPreRun(cmd *cobra.Command, _ []string) error {
	if noInteractive {
		interactive.Disable()
	}
	log.SetVerbose(verbosity)
	slog.Debug("Running PersistentPreRunE", "command", "root")
	hooks.SetTrustMode(hooks.ParseTrustMode(trustHooks))

	isConfigCmd := cmd.Name() == "config" || (cmd.Parent() != nil && cmd.Parent().Name() == "config")

	prefs, err := config.LoadPreferences(cmd.Flags(), cfgFile)
	if err != nil {
		if isConfigCmd {
			slog.Warn(err.Error())
			return nil
		}
		return err
	}

	if prefs.NoColor.Value {
		os.Setenv("NO_COLOR", "1")
		lipgloss.Writer.Profile = colorprofile.Ascii
	}

	if prefs.Theme.Value != "" {
		t, err := theme.LoadThemeFromFile(prefs.Theme.Value)
		if err != nil {
			slog.Warn("Using default theme", "error", err)
		}
		theme.SetTheme(t)
	}

	if isConfigCmd {
		return nil
	}

	if err := config.Initialize(prefs, onlyChanges, workingDir); err != nil {
		slog.Debug(fmt.Sprintf("Run context initialization skipped: %v", err))
	}

	onStopProfiling = profilingInit()

	return nil
}

func NewCommand() *cobra.Command {
	cobra.EnableTraverseRunHooks = true

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.gws/config.yaml)")
	rootCmd.PersistentFlags().String(config.KeyTheme, "", "theme file")
	rootCmd.PersistentFlags().Int(config.KeyParallel, gws2.DefaultParallel, "number of parallel operations")
	rootCmd.PersistentFlags().String(config.KeyFormat, config.DefaultFormat, "output format (text, json, yaml)")
	rootCmd.PersistentFlags().Bool(config.KeyNoColor, false, "disable colored output")
	rootCmd.PersistentFlags().BoolVarP(&onlyChanges, "only-changes", "c", false, "show only repositories with changes")
	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&trustHooks, "trust-hooks", "ask", "trust mode for local hooks: ask, all, skip")
	rootCmd.PersistentFlags().Bool(config.KeyStopOnError, false, "stop execution on first error")
	rootCmd.PersistentFlags().StringVarP(&workingDir, "working-dir", "D", "", "set working directory for the command. If not set the current directory is used")
	rootCmd.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "disable interactive UI and prompts (also enabled by CI=true or "+interactive.EnvNoInteractive+"=true)")

	// profiling
	applyProfilingFlags(rootCmd)

	return rootCmd
}

func GetRunContext() *config.RunContext {
	return config.GetRunContext()
}
