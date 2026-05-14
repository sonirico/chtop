package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderBar produces a width-wide horizontal bar with the first
// progress*width cells filled in fillColor and the rest in emptyColor. Uses
// the unicode block elements so partial cells look smooth.
func renderBar(progress float64, width int, fillColor, emptyColor lipgloss.TerminalColor) string {
	if width <= 0 {
		return ""
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	full := int(progress * float64(width))
	rem := progress*float64(width) - float64(full)
	partials := []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}
	partialIdx := int(rem * 8)
	if partialIdx > 7 {
		partialIdx = 7
	}

	filled := strings.Repeat("█", full)
	if full < width && partialIdx > 0 {
		filled += partials[partialIdx]
	}
	emptyN := width - full
	if partialIdx > 0 && full < width {
		emptyN--
	}
	if emptyN < 0 {
		emptyN = 0
	}
	empty := strings.Repeat("░", emptyN)
	return lipgloss.NewStyle().Foreground(fillColor).Render(filled) +
		lipgloss.NewStyle().Foreground(emptyColor).Render(empty)
}

// progressColor returns a foreground colour that fades from red (just
// started) through yellow (mid) to green (almost done). Helps spot stuck
// merges and ones about to finish without reading the percentage.
func progressColor(progress float64) lipgloss.TerminalColor {
	switch {
	case progress < 0.33:
		return colorErr
	case progress < 0.75:
		return colorPrimary
	default:
		return colorOK
	}
}
