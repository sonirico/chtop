package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpView is the scrollable cheat sheet shown via `?` or `:help`.
type HelpView struct {
	app      *App
	viewport viewport.Model
}

func newHelpView(app *App) *HelpView {
	vp := viewport.New(80, 20)
	vp.SetContent(helpContent())
	return &HelpView{app: app, viewport: vp}
}

func (v *HelpView) SetSize(w, h int) {
	if w > 0 {
		v.viewport.Width = w
	}
	if h > 2 {
		v.viewport.Height = h
	}
}

func (v *HelpView) Init() tea.Cmd { return nil }

func (v *HelpView) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return cmd
}

func (v *HelpView) View() string {
	return v.viewport.View()
}

func helpContent() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	dim := lipgloss.NewStyle().Foreground(colorMuted)
	section := func(title string) string {
		return "\n" + bold.Render(title) + "\n"
	}
	row := func(k, desc string) string {
		return "  " + bold.Render(padRight(k, 14)) + dim.Render(desc) + "\n"
	}

	var b strings.Builder
	b.WriteString(bold.Render("chtop - keymap"))
	b.WriteString("\n")

	b.WriteString(section("Navigation"))
	b.WriteString(row(":", "open command bar"))
	b.WriteString(row("/", "filter rows (esc to clear, enter to keep)"))
	b.WriteString(row("up/down", "move cursor"))
	b.WriteString(row("r", "refresh"))
	b.WriteString(row("esc", "back"))
	b.WriteString(row("?", "toggle help"))
	b.WriteString(row("q ctrl+c", "quit"))

	b.WriteString(section("Commands (after `:`)"))
	b.WriteString(row("tables, t", "tables list"))
	b.WriteString(row("processes, p", "running queries"))
	b.WriteString(row("clusters, c", "cluster topology"))
	b.WriteString(row("replicas, R", "replication status"))
	b.WriteString(row("merges, m", "merges + mutations in flight"))
	b.WriteString(row("querylog, ql", "recent finished/failed queries"))
	b.WriteString(row("help, ?", "this help"))

	b.WriteString(section("Query log view"))
	b.WriteString(row("e / enter", "open EXPLAIN for the selected query"))
	b.WriteString(row("quit, q", "quit"))

	b.WriteString(section("Processes view"))
	b.WriteString(row("k", "kill the selected query (asks y/N)"))

	return b.String()
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
