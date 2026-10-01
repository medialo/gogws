package configcmd

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/medialo/gogws/internal/config"
	"github.com/medialo/gogws/internal/ui/cli"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage gogws configuration",
		Long:  `View and manage gogws user configuration stored in ~/.gws/config.yaml`,
		RunE:  runConfigShow,
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
		Long:  `Get a specific configuration value by key.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	}
}

func newSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value.

Available keys:
  provider-cache-ttl    How long a discovered provider workspace is kept before --refresh-providers re-reads it (e.g. 12h)
  trusted-workspaces    Deprecated, no longer grants trust to hooks`,
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

func runConfigShow(cmd *cobra.Command, args []string) error {
	slog.Debug("Loading user configuration...")

	resolved, err := config.LoadUserConfigResolved()
	if err != nil {
		return fmt.Errorf("failed to load user config: %w", err)
	}

	configPath, _ := config.GetUserConfigPath()

	renderer := cli.NewRenderer()

	lipgloss.Println(renderer.RenderHeader("GOGWS Configuration"))
	lipgloss.Println()
	lipgloss.Printf("  File: %s\n\n", configPath)

	if len(resolved.TrustedWorkspaces.Value) > 0 {
		lipgloss.Println(renderer.RenderConfigValue("trusted-workspaces", resolved.TrustedWorkspaces.Value, string(resolved.TrustedWorkspaces.Source)))
	} else {
		lipgloss.Println(renderer.RenderConfigValue("trusted-workspaces", "(none)", string(resolved.TrustedWorkspaces.Source)))
	}
	lipgloss.Println(renderer.RenderConfigValue("provider-cache-ttl", resolved.ProviderCacheTTL.Value, string(resolved.ProviderCacheTTL.Source)))

	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	slog.Debug("Getting config value", "key", key)

	resolved, err := config.LoadUserConfigResolved()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "trusted-workspaces":
		if len(resolved.TrustedWorkspaces.Value) == 0 {
			lipgloss.Printf("(none) (source: %s)\n", resolved.TrustedWorkspaces.Source)
		} else {
			lipgloss.Printf("(source: %s)\n", resolved.TrustedWorkspaces.Source)
			for _, ws := range resolved.TrustedWorkspaces.Value {
				lipgloss.Printf("  - %s\n", ws)
			}
		}
	case "provider-cache-ttl":
		lipgloss.Printf("%s (source: %s)\n", resolved.ProviderCacheTTL.Value, resolved.ProviderCacheTTL.Source)
	default:
		return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n  %s",
			key, strings.Join(config.GetAvailableConfigKeys(), "\n  "))
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	valueStr := args[1]

	slog.Debug("Setting config value", "key", key, "value", valueStr)

	switch key {
	case "trusted-workspaces":
		if err := config.AddTrustedWorkspace(valueStr); err != nil {
			return fmt.Errorf("failed to add trusted workspace: %w", err)
		}
		renderer := cli.NewRenderer()
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("Added trusted workspace: %s", valueStr)))
		return nil
	case "provider-cache-ttl":
		if _, err := time.ParseDuration(valueStr); err != nil {
			return fmt.Errorf("invalid duration %q for provider-cache-ttl (e.g. 30m, 12h): %w", valueStr, err)
		}
		if err := config.SetUserConfigValue(key, valueStr); err != nil {
			return fmt.Errorf("failed to set provider-cache-ttl: %w", err)
		}
		renderer := cli.NewRenderer()
		lipgloss.Println(renderer.RenderSuccess(fmt.Sprintf("provider-cache-ttl set to %s", valueStr)))
		return nil
	default:
		return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n  %s",
			key, strings.Join(config.GetAvailableConfigKeys(), "\n  "))
	}
}

func runConfigList(cmd *cobra.Command, args []string) error {
	renderer := cli.NewRenderer()

	lipgloss.Println(renderer.RenderHeader("Available Configuration Keys"))
	lipgloss.Println()

	keys := config.GetAvailableConfigKeys()
	for _, key := range keys {
		lipgloss.Printf("  %s\n", key)

		switch key {
		case "trusted-workspaces":
			lipgloss.Printf("    type: list of paths\n")
			lipgloss.Printf("    desc: Deprecated, no longer grants trust. Local hooks are trusted per file (path + sha256) in ~/.gws/%s\n", config.TrustedHooksFile)
		case "provider-cache-ttl":
			lipgloss.Printf("    type: duration (default %s)\n", config.DefaultProviderCacheTTL)
			lipgloss.Printf("    desc: How long a discovered provider workspace is kept before --refresh-providers re-reads it\n")
		}
		lipgloss.Println()
	}

	return nil
}
