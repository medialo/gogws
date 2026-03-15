package help

import (
	"fmt"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

func renderWithGlamour(text string) string {
	if text == "" {
		return ""
	}

	renderer, err := glamour.NewTermRenderer(
		//glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return text
	}

	renderedText, err := renderer.Render(text)
	if err != nil {
		return text
	}

	return strings.TrimSpace(renderedText)
}

func HelpFunc(command *cobra.Command, args []string) {

	if true {
		helpTemplated := command.UseLine()
		fmt.Println(renderWithGlamour(helpTemplated))
	}

	originalLong := command.Long
	originalExample := command.Example

	//if command.Long != "" {
	//	command.Long = renderWithGlamour(command.Long)
	//}
	//if command.Example != "" {
	//	command.Example = renderWithGlamour(command.Example)
	//}

	//var coreCommands []string
	//var additionalCommands []string
	//for _, c := range command.Commands() {
	//	if c.Short == "" {
	//		continue
	//	}
	//	if !c.IsAvailableCommand() {
	//		continue
	//	}
	//
	//	s := rpad(c.Name()+":", c.NamePadding()) + c.Short
	//	if _, ok := c.Annotations["IsCore"]; ok {
	//		coreCommands = append(coreCommands, s)
	//	} else {
	//		additionalCommands = append(additionalCommands, s)
	//	}
	//}

	type helpEntry struct {
		Title string
		Body  string
	}

	var helpEntries []helpEntry
	if command.Long != "" {
		helpEntries = append(helpEntries, helpEntry{"", command.Long})
	} else if command.Short != "" {
		helpEntries = append(helpEntries, helpEntry{"", command.Short})
	}
	helpEntries = append(helpEntries, helpEntry{"USAGE", command.UseLine()})
	if len(command.Aliases) > 0 {
		helpEntries = append(helpEntries, helpEntry{"ALIASES", strings.Join(command.Aliases, ", ")})
	}
	//if len(coreCommands) > 0 {
	//	helpEntries = append(helpEntries, helpEntry{"CORE COMMANDS", strings.Join(coreCommands, "\n")})
	//}
	//if len(additionalCommands) > 0 {
	//	helpEntries = append(helpEntries, helpEntry{"ADDITIONAL COMMANDS", strings.Join(additionalCommands, "\n")})
	//}

	flagUsages := command.LocalFlags().FlagUsages()
	if flagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"FLAGS", flagUsages})
	}
	inheritedFlagUsages := command.InheritedFlags().FlagUsages()
	if inheritedFlagUsages != "" {
		helpEntries = append(helpEntries, helpEntry{"INHERITED FLAGS", inheritedFlagUsages})
	}
	if _, ok := command.Annotations["help:arguments"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ARGUMENTS", command.Annotations["help:arguments"]})
	}
	if command.Example != "" {
		helpEntries = append(helpEntries, helpEntry{"EXAMPLES", command.Example})
	}
	if _, ok := command.Annotations["help:environment"]; ok {
		helpEntries = append(helpEntries, helpEntry{"ENVIRONMENT VARIABLES", command.Annotations["help:environment"]})
	}
	helpEntries = append(helpEntries, helpEntry{"LEARN MORE", `
Use 'glab <command> <subcommand> --help' for more information about a command.`})
	if _, ok := command.Annotations["help:feedback"]; ok {
		helpEntries = append(helpEntries, helpEntry{"FEEDBACK", command.Annotations["help:feedback"]})
	}

	out := command.OutOrStdout()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241"))

	for _, e := range helpEntries {
		if e.Title != "" {
			// If there is a title, add indentation to each line in the body
			fmt.Fprintln(out, titleStyle.Render(e.Title))
			fmt.Fprintln(out, strings.Trim(e.Body, "\r\n"))
			//fmt.Fprintln(out, utils.Indent(strings.Trim(e.Body, "\r\n"), "  ")
		} else {
			// If there is no title print the body as is
			fmt.Fprintln(out, e.Body)
		}
		fmt.Fprintln(out)
	}

	command.Long = originalLong
	command.Example = originalExample
}
