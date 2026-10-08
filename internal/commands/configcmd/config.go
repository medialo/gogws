package configcmd

import (
	"fmt"
	"log/slog"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gogws configuration",
		Long: `View and manage gogws user configuration stored in ~/.gws/config.yaml.

Each value is resolved in this order: flag > environment variable (GOGWS_*) > config file > default.`,
		RunE: runConfigShow,
	}

	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newListCommand())

	return cmd
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  `Get the effective value of a configuration key and where it comes from.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	}
}

func newSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value in ~/.gws/config.yaml.

Run "gogws config list" to see the available keys.`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available configuration keys",
		RunE:  runConfigList,
	}
}

func loadPreferences(cmd *cobra.Command) (*config.Preferences, error) {
	configFile, _ := cmd.Flags().GetString("config")
	prefs, err := config.LoadPreferences(cmd.Flags(), configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	return prefs, nil
}

func unknownKeyError(key string) error {
	names := make([]string, 0, len(config.PreferenceKeys))
	for _, k := range config.PreferenceKeys {
		names = append(names, k.Name)
	}
	return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n  %s", key, strings.Join(names, "\n  "))
}

func displayValue(prefs *config.Preferences, key string) (any, config.ConfigSource, bool) {
	switch key {
	case config.KeyParallel:
		return prefs.Parallel.Value, prefs.Parallel.Source, true
	case config.KeyFormat:
		return prefs.Format.Value, prefs.Format.Source, true
	case config.KeyTheme:
		return formatValue(prefs.Theme.Value), prefs.Theme.Source, true
	case config.KeyNoColor:
		return prefs.NoColor.Value, prefs.NoColor.Source, true
	case config.KeyStopOnError:
		return prefs.StopOnError.Value, prefs.StopOnError.Source, true
	case config.KeyProviderCacheTTL:
		return prefs.ProviderCacheTTL.Value, prefs.ProviderCacheTTL.Source, true
	case config.KeyUpdateCheck:
		return prefs.UpdateCheck.Value, prefs.UpdateCheck.Source, true
	case config.KeyTrustedWorkspaces:
		return formatValue(prefs.TrustedWorkspaces.Value), prefs.TrustedWorkspaces.Source, true
	default:
		return nil, "", false
	}
}

func formatValue(value any) any {
	switch v := value.(type) {
	case string:
		if v == "" {
			return "(none)"
		}
	case []string:
		if len(v) == 0 {
			return "(none)"
		}
	}
	return value
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	slog.Debug("Loading user configuration...")

	prefs, err := loadPreferences(cmd)
	if err != nil {
		return err
	}

	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath, _ = config.GetUserConfigPath()
	}

	renderer := cli.NewRenderer()

	lipgloss.Println(renderer.RenderHeader("GOGWS Configuration"))
	lipgloss.Println()
	lipgloss.Printf("  File: %s\n\n", configPath)

	for _, k := range config.PreferenceKeys {
		value, source, _ := displayValue(prefs, k.Name)
		lipgloss.Println(renderer.RenderConfigValue(k.Name, value, string(source)))
	}

	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	slog.Debug("Getting config value", "key", key)

	prefs, err := loadPreferences(cmd)
	if err != nil {
		return err
	}

	value, source, ok := displayValue(prefs, key)
	if !ok {
		return unknownKeyError(key)
	}

	if list, isList := value.([]string); isList {
		lipgloss.Printf("(source: %s)\n", source)
		for _, item := range list {
			lipgloss.Printf("  - %s\n", item)
		}
		return nil
	}

	lipgloss.Printf("%v (source: %s)\n", value, source)
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	valueStr := args[1]

	slog.Debug("Setting config value", "key", key, "value", valueStr)

	if !config.IsPreferenceKey(key) {
		return unknownKeyError(key)
	}

	renderer := cli.NewRenderer()

	if key == config.KeyTrustedWorkspaces {
		if err := config.AddTrustedWorkspace(valueStr); err != nil {
			return fmt.Errorf("failed to add trusted workspace: %w", err)
		}
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added trusted workspace: %s", valueStr)))
		return nil
	}

	if err := config.SetPreference(key, valueStr); err != nil {
		return err
	}
	lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("%s set to %s", key, valueStr)))
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	renderer := cli.NewRenderer()

	lipgloss.Println(renderer.RenderHeader("Available Configuration Keys"))
	lipgloss.Println()

	for _, k := range config.PreferenceKeys {
		lipgloss.Printf("  %s\n", k.Name)
		lipgloss.Printf("    type: %s (default %v)\n", k.Type, formatValue(k.Default))
		lipgloss.Printf("    env:  %s\n", config.EnvVarName(k.Name))
		lipgloss.Printf("    desc: %s\n", k.Description)
		lipgloss.Println()
	}

	return nil
}
