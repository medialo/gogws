package root

import (
	"fmt"
	"gogws/internal/config"
	"gogws/internal/hooks"
	"gogws/internal/log"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	themeFile   string
	parallel    int
	format      string
	noColor     bool
	onlyChanges bool
	verbose     bool
	trustHooks  string
	stopOnError bool
	workingDir  string
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
	log.SetVerbose(verbose)
	slog.Debug("Running PersistentPreRunE", "command", "root")
	hooks.SetTrustMode(hooks.ParseTrustMode(trustHooks))

	if cmd.Name() == "config" {
		return nil
	}

	if err := config.Initialize(); err != nil {
		slog.Debug(fmt.Sprintf("Config initialization skipped: %v", err))
	}

	if config.IsInitialized() {
		config.ApplyFlags(themeFile, parallel, format, noColor, onlyChanges, stopOnError, workingDir)
	}

	return nil
}

func NewCommand() *cobra.Command {
	cobra.OnInitialize(initConfig)

	cobra.EnableTraverseRunHooks = true

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: $HOME/.config/gogws/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&themeFile, "theme", "", "theme file")
	rootCmd.PersistentFlags().IntVar(&parallel, "parallel", 0, "number of parallel operations (default: 5)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "text", "output format (text, json, yaml)")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	rootCmd.PersistentFlags().BoolVar(&onlyChanges, "only-changes", false, "show only repositories with changes")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().StringVar(&trustHooks, "trust-hooks", "ask", "trust mode for local hooks: ask, all, skip")
	rootCmd.PersistentFlags().BoolVar(&stopOnError, "stop-on-error", false, "stop execution on first error")
	rootCmd.PersistentFlags().StringVarP(&workingDir, "working-dir", "D", "", "set working directory for the command. Is not set the current directory is used")

	viper.BindPFlag("theme", rootCmd.PersistentFlags().Lookup("theme"))
	viper.BindPFlag("parallel", rootCmd.PersistentFlags().Lookup("parallel"))
	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	viper.BindPFlag("working_dir", rootCmd.PersistentFlags().Lookup("working-dir"))

	return rootCmd
}

func GetConfig() *config.Config {
	return config.GetConfig()
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home + "/.config/gogws")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("GOGWS")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("verbose") {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}
}
