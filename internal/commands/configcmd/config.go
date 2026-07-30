package configcmd

import (
	"fmt"
	"log/slog"
	"strings"

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
  trusted-workspaces    List of trusted workspace paths for hooks`,
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

	fmt.Println(renderer.RenderHeader("GOGWS Configuration"))
	fmt.Println()
	fmt.Printf("  File: %s\n\n", configPath)

	if len(resolved.TrustedWorkspaces.Value) > 0 {
		fmt.Println(renderer.RenderConfigValue("trusted-workspaces", resolved.TrustedWorkspaces.Value, string(resolved.TrustedWorkspaces.Source)))
	} else {
		fmt.Println(renderer.RenderConfigValue("trusted-workspaces", "(none)", string(resolved.TrustedWorkspaces.Source)))
	}

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
			fmt.Printf("(none) (source: %s)\n", resolved.TrustedWorkspaces.Source)
		} else {
			fmt.Printf("(source: %s)\n", resolved.TrustedWorkspaces.Source)
			for _, ws := range resolved.TrustedWorkspaces.Value {
				fmt.Printf("  - %s\n", ws)
			}
		}
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
		fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Added trusted workspace: %s", valueStr)))
		return nil
	default:
		return fmt.Errorf("unknown configuration key: %s\n\nAvailable keys:\n  %s",
			key, strings.Join(config.GetAvailableConfigKeys(), "\n  "))
	}
}

func runConfigList(cmd *cobra.Command, args []string) error {
	renderer := cli.NewRenderer()

	fmt.Println(renderer.RenderHeader("Available Configuration Keys"))
	fmt.Println()

	keys := config.GetAvailableConfigKeys()
	for _, key := range keys {
		fmt.Printf("  %s\n", key)

		switch key {
		case "trusted-workspaces":
			fmt.Printf("    type: list of paths\n")
			fmt.Printf("    desc: Workspace paths where local hooks are trusted\n")
		}
		fmt.Println()
	}

	return nil
}
