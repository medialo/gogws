package dev

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "dev",
		Short:  "Development and testing commands",
		Long:   `Hidden commands for development, testing, and debugging purposes.`,
		Hidden: true,
	}

	cmd.AddCommand(NewGenerateCommand())
	cmd.AddCommand(NewPingCommand())

	return cmd
}
