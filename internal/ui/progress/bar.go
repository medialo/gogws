package progress

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type Bar struct {
	width   int
	current int
	total   int
	prefix  string
}

func New(total int, width int) *Bar {
	return &Bar{
		width:   width,
		total:   total,
		current: 0,
	}
}

func (b *Bar) SetPrefix(prefix string) {
	b.prefix = prefix
}

func (b *Bar) Increment() {
	if b.current < b.total {
		b.current++
	}
}

func (b *Bar) SetCurrent(current int) {
	if current <= b.total {
		b.current = current
	}
}

func (b *Bar) Render() string {
	if b.total == 0 {
		return ""
	}

	percentage := float64(b.current) / float64(b.total)
	filled := int(float64(b.width) * percentage)
	empty := b.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	percentText := fmt.Sprintf("%.0f%%", percentage*100)
	countText := fmt.Sprintf("(%d/%d)", b.current, b.total)

	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	percentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))

	result := ""
	if b.prefix != "" {
		result += b.prefix + " "
	}
	result += progressStyle.Render(bar) + " " + percentStyle.Render(percentText) + " " + countText

	return result
}

func (b *Bar) Complete() bool {
	return b.current >= b.total
}

func (b *Bar) Percentage() float64 {
	if b.total == 0 {
		return 0
	}
	return float64(b.current) / float64(b.total) * 100
}
