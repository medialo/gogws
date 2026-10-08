package prompt

import (
	"errors"
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
	"github.com/medialo/gogws/internal/interactive"
)

var ErrNotInteractive = errors.New("an interactive terminal is required")

func KeyMap() *huh.KeyMap {
	keyMap := huh.NewDefaultKeyMap()
	keyMap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	return keyMap
}

func Available() bool {
	return interactive.CanPrompt()
}

func Require(what string) error {
	if Available() {
		return nil
	}
	return fmt.Errorf("%s: %w; pass the value as an argument or flag", what, ErrNotInteractive)
}

func RunField(field huh.Field) error {
	if err := Require("missing input"); err != nil {
		return err
	}
	form := huh.NewForm(huh.NewGroup(field)).WithShowHelp(false).WithKeyMap(KeyMap())
	return form.Run()
}

func RunForm(what string, form *huh.Form) error {
	if err := Require(what); err != nil {
		return err
	}
	return form.WithKeyMap(KeyMap()).Run()
}
