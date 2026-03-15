package log

import (
	"log/slog"
	"os"

	"charm.land/lipgloss/v2"
	"charm.land/log/v2"
	"github.com/dpotapov/slogpfx"
)

var (
	baseHandler *log.Logger
	logger      *slog.Logger
)

func init() {
	styles := log.DefaultStyles()
	styles.Prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	styles.Levels[log.DebugLevel].
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("0"))

	baseHandler = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})
	baseHandler.SetStyles(styles)
	baseHandler.SetLevel(log.InfoLevel)
	baseHandler.SetReportCaller(true)

	prefixedHandler := slogpfx.NewHandler(baseHandler, &slogpfx.HandlerOptions{
		PrefixKeys: []string{"context", "workspace", "project"},
		//PrefixFormatter: func(prefixes []slog.Value) string {
		//	return ""
		//},
	})

	logger = slog.New(prefixedHandler)
	slog.SetDefault(logger)
}

func SetVerbose(v bool) {
	if v {
		baseHandler.SetLevel(log.DebugLevel)
	} else {
		baseHandler.SetLevel(log.InfoLevel)
	}
}
