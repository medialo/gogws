package commands

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"charm.land/fang/v2"
	"github.com/medialo/gogws/internal/commands/add"
	"github.com/medialo/gogws/internal/commands/check"
	"github.com/medialo/gogws/internal/commands/clone"
	"github.com/medialo/gogws/internal/commands/configcmd"
	"github.com/medialo/gogws/internal/commands/dev"
	"github.com/medialo/gogws/internal/commands/doctor"
	"github.com/medialo/gogws/internal/commands/fetch"
	"github.com/medialo/gogws/internal/commands/ff"
	"github.com/medialo/gogws/internal/commands/initcmd"
	"github.com/medialo/gogws/internal/commands/root"
	"github.com/medialo/gogws/internal/commands/search"
	"github.com/medialo/gogws/internal/commands/status"
	"github.com/medialo/gogws/internal/commands/update"
	"github.com/medialo/gogws/internal/commands/version"
	"github.com/spf13/cobra"
)

func Execute() error {
	slog.Debug("Starting gogws")
	rootCmd := root.NewCommand()

	statusCmd := status.NewCommand(root.GetRunContext)

	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(clone.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(fetch.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(ff.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(check.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(initcmd.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(update.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(add.NewCommand(root.GetRunContext))
	rootCmd.AddCommand(configcmd.NewCommand())
	rootCmd.AddCommand(dev.NewCommand())
	rootCmd.AddCommand(doctor.NewDoctorCommand(root.GetRunContext))
	rootCmd.AddCommand(search.NewCommand(root.GetRunContext))

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

	return fang.Execute(context.Background(), rootCmd, fang.WithVersion(version.Current()), fang.WithCommit(version.Commit))
}
