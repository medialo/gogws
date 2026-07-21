package log

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"charm.land/log/v2"
	"golang.org/x/term"
)

var (
	baseHandler *log.Logger
	logger      *slog.Logger
	f           *os.File
)

const (
	DebugLevel1 = slog.LevelDebug
	DebugLevel2 = slog.Level(-10)
	DebugLevel3 = slog.Level(-14)
)

func init() {

	baseHandler = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
	})

	baseHandler.SetLevel(log.InfoLevel)
	baseHandler.SetReportCaller(true)

	logger = slog.New(baseHandler)
	slog.SetDefault(logger)
}

func SetVerbose(verbosity int) {
	switch verbosity {
	case 1:
		baseHandler.SetLevel(log.Level(DebugLevel1))
	case 2:
		baseHandler.SetLevel(log.Level(DebugLevel2))
	case 3:
		baseHandler.SetLevel(log.Level(DebugLevel3))
	default:
		baseHandler.SetLevel(log.InfoLevel)
	}

	if verbosity > 0 {
		isInteractive := term.IsTerminal(int(os.Stdout.Fd())) // todo centralized isInteractive value to allow --interactive or --no

		if isInteractive {
			slog.Debug("You are running gogws in verbose mode and with an interactive terminal. Logs will be written to a file.")
			currTime := time.Now()
			epoch := currTime.Unix()
			formatedTime := currTime.Format("2006-01-02")
			fileName := "gogws_" + formatedTime + "_" + strconv.FormatInt(epoch, 10) + ".log"
			var err error
			f, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				panic(err)
			}
			baseHandler.SetOutput(f)
		}
	}
}

func Close() {
	if f != nil {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}
}
