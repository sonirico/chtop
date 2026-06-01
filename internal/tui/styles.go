package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

// ClickHouse-ish palette. Yellow as primary (headers, status). The cursor /
// selected row goes red so it pops against the yellow body, matching the
// brand's accent dot.
var (
	colorPrimary   = lipgloss.AdaptiveColor{Light: "#A77E12", Dark: "#FFCC00"}
	colorPrimaryHi = lipgloss.AdaptiveColor{Light: "#C9971F", Dark: "#FFE066"}
	colorCursor    = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#EF4444"}
	colorMuted     = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	colorOK        = lipgloss.AdaptiveColor{Light: "#10B981", Dark: "#34D399"}
	colorErr       = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}
	colorBorder    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"}
	colorSelFG     = lipgloss.Color("#FFFFFF")
	colorBarBG     = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#1F1F1F"}
)

// Sparkline intensity bands: low values stay calm green, mid go yellow, high
// turn red — the btop-style "this metric is hot" cue.
var (
	sparkLow  = lipgloss.NewStyle().Foreground(colorOK)
	sparkMid  = lipgloss.NewStyle().Foreground(colorPrimary)
	sparkHigh = lipgloss.NewStyle().Foreground(colorErr)
)

// Styles holds the lip gloss styles used across the TUI.
type Styles struct {
	Header     lipgloss.Style
	HeaderKey  lipgloss.Style
	Footer     lipgloss.Style
	StatusOK   lipgloss.Style
	StatusErr  lipgloss.Style
	CommandBar lipgloss.Style
	Error      lipgloss.Style
	Title      lipgloss.Style
}

func newStyles() Styles {
	return Styles{
		Header:     lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Padding(0, 1),
		HeaderKey:  lipgloss.NewStyle().Foreground(colorMuted),
		Footer:     lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1),
		StatusOK:   lipgloss.NewStyle().Foreground(colorOK),
		StatusErr:  lipgloss.NewStyle().Foreground(colorErr),
		CommandBar: lipgloss.NewStyle().Background(colorBarBG).Padding(0, 1),
		Error:      lipgloss.NewStyle().Foreground(colorErr).Bold(true),
		Title:      lipgloss.NewStyle().Bold(true).Foreground(colorPrimary),
	}
}

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		Foreground(colorPrimaryHi).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(colorSelFG).
		Background(colorCursor).
		Bold(true)
	return s
}
