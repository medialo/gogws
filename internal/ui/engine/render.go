package engine

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

//type Engine interface {
//	Total() int
//	Run() error
//}

type TeaEngineRenderer struct {
}

func (m TeaEngineRenderer) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m TeaEngineRenderer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//	switch msg := msg.(type) {
	//
	//	// Is it a key press?
	//	case tea.KeyPressMsg:
	//
	//		// Cool, what was the actual key pressed?
	//		switch msg.String() {
	//
	//		// These keys should exit the program.
	//		case "ctrl+c", "q":
	//			return m, tea.Quit
	//
	//		// The "up" and "k" keys move the cursor up
	//		case "up", "k":
	//			if m.cursor > 0 {
	//				m.cursor--
	//			}
	//
	//		// The "down" and "j" keys move the cursor down
	//		case "down", "j":
	//			if m.cursor < len(m.choices)-1 {
	//				m.cursor++
	//			}
	//
	//		// The "enter" key and the space bar toggle the selected state
	//		// for the item that the cursor is pointing at.
	//		case "enter", "space":
	//			_, ok := m.selected[m.cursor]
	//			if ok {
	//				delete(m.selected, m.cursor)
	//			} else {
	//				m.selected[m.cursor] = struct{}{}
	//			}
	//		}
	//	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m TeaEngineRenderer) View() tea.View {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	//for i, choice := range m.choices {
	//
	//	// Is the cursor pointing at this choice?
	//	cursor := " " // no cursor
	//	if m.cursor == i {
	//		cursor = ">" // cursor!
	//	}
	//
	//	// Is this choice selected?
	//	checked := " " // not selected
	//	if _, ok := m.selected[i]; ok {
	//		checked = "x" // selected!
	//	}
	//
	//	// Render the row
	//	s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	//}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return tea.NewView(s)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
