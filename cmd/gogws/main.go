package main

import (
	"os"

	"github.com/medialo/gogws/internal/commands"
	"github.com/medialo/gogws/internal/log"
)

func main() {
	defer log.Close()
	if err := commands.Execute(); err != nil {
		//fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
