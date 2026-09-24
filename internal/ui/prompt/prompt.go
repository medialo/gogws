package prompt

import (
	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
)

func KeyMap() *huh.KeyMap {
	keyMap := huh.NewDefaultKeyMap()
	keyMap.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	return keyMap
}

func RunField(field huh.Field) error {
	form := huh.NewForm(huh.NewGroup(field)).WithShowHelp(false).WithKeyMap(KeyMap())
	return form.Run()
}
