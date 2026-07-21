package commands

import (
	"context"
	"gogws/internal/commands/add"
	"gogws/internal/commands/check"
	"gogws/internal/commands/clone"
	"gogws/internal/commands/configcmd"
	"gogws/internal/commands/dev"
	"gogws/internal/commands/doctor"
	"gogws/internal/commands/fetch"
	"gogws/internal/commands/ff"
	"gogws/internal/commands/initcmd"
	"gogws/internal/commands/root"
	"gogws/internal/commands/status"
	"gogws/internal/commands/update"
	"gogws/internal/commands/version"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

func Execute() error {
	slog.Debug("Starting gogws")
	rootCmd := root.NewCommand()

	statusCmd := status.NewCommand(root.GetConfig)

	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(clone.NewCommand(root.GetConfig))
	rootCmd.AddCommand(fetch.NewCommand(root.GetConfig))
	rootCmd.AddCommand(ff.NewCommand(root.GetConfig))
	rootCmd.AddCommand(check.NewCommand(root.GetConfig))
	rootCmd.AddCommand(initcmd.NewCommand(root.GetConfig))
	rootCmd.AddCommand(update.NewCommand(root.GetConfig))
	rootCmd.AddCommand(add.NewCommand())
	rootCmd.AddCommand(configcmd.NewCommand())
	rootCmd.AddCommand(dev.NewCommand())
	rootCmd.AddCommand(doctor.NewDoctorCommand(root.GetConfig))

	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return statusCmd.RunE(cmd, args)
	}

	defer root.StopProfiling()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		root.StopProfiling()
		os.Exit(1)
	}()

	return fang.Execute(context.Background(), rootCmd)
}
