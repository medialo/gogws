package main

import (
	"os"

	"github.com/medialo/gogws/internal/commands"
	"github.com/medialo/gogws/internal/log"
)

func main() {
	defer log.Close()
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
