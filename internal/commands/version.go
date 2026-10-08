package commands

import (
	"fmt"
	"io"
	"runtime/debug"
)

var (
	Version = "dev"
	Commit  = ""
)

func Current() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}

func Print(w io.Writer) error {
	_, err := fmt.Fprintln(w, Current())
	return err
}
