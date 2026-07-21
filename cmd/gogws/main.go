package main

import (
	"gogws/internal/commands"
	"gogws/internal/log"
	"os"
)

func main() {
	defer log.Close()
	if err := commands.Execute(); err != nil {
		//fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
