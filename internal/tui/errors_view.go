package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sonirico/chtop/pkg/ch"
)

// ErrorsView surfaces system.errors. Errors that have grown since the
// previous snapshot show their count in red, so a sudden spike is visible
// without reading every line.
type ErrorsView struct {
	app     *App
	table   table.Model
	current []ch.ErrorInfo
	prev    map[errorKey]uint64
	errored bool
}

// errorKey is the unique-by-(name, remote) tuple system.errors uses.
type errorKey struct {
	Name   string
	Remote bool
}

func newErrorsView(app *App) *ErrorsView {
	tbl := table.New(
		table.WithColumns(errorsColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &ErrorsView{app: app, table: tbl, prev: map[errorKey]uint64{}}
}

func (v *ErrorsView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(errorsColumns(w))
	}
}

func errorsColumns(width int) []table.Column {
	const (
		codeW   = 5
		countW  = 9
		seenW   = 20
		remoteW = 6
		gutter  = 2
	)
	fixed := codeW + countW + seenW + remoteW + gutter*5
	// Split remaining between NAME and LAST MESSAGE.
	rest := width - fixed
	if rest < 40 {
		rest = 40
	}
	nameW := rest / 3
	if nameW < 18 {
		nameW = 18
	}
	msgW := rest - nameW
	if msgW < 20 {
		msgW = 20
	}
	return []table.Column{
		{Title: "NAME", Width: nameW},
		{Title: "CODE", Width: codeW},
		{Title: "COUNT", Width: countW},
		{Title: "LAST SEEN", Width: seenW},
		{Title: "WHERE", Width: remoteW},
		{Title: "MESSAGE", Width: msgW},
	}
}

func (v *ErrorsView) Init() tea.Cmd {
	return v.load()
}

func (v *ErrorsView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case errorsLoadedMsg:
		v.current = m.errors
		v.errored = false
		v.refreshRows()
		v.snapshot()
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewErrors || v.errored {
			return nil
		}
		return v.load()
	case tea.KeyMsg:
		if m.String() == "r" {
			v.errored = false
			return v.load()
		}
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return cmd
}

func (v *ErrorsView) View() string {
	if !v.errored && len(v.current) == 0 {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("no errors recorded") + "\n"
	}
	return v.table.View()
}

func (v *ErrorsView) refreshRows() {
	rows := make([]table.Row, 0, len(v.current))
	hot := lipgloss.NewStyle().Foreground(colorErr).Bold(true)
	for _, e := range v.current {
		row := errorRow(e)
		prev := v.prev[errorKey{Name: e.Name, Remote: e.Remote}]
		if isGrowing(prev, e.Value) {
			// Re-color the count cell to highlight growth at a glance.
			row[2] = hot.Render(row[2])
		}
		rows = append(rows, table.Row(row))
	}
	v.table.SetRows(rows)
}

func (v *ErrorsView) snapshot() {
	v.prev = make(map[errorKey]uint64, len(v.current))
	for _, e := range v.current {
		v.prev[errorKey{Name: e.Name, Remote: e.Remote}] = e.Value
	}
}

func (v *ErrorsView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		es, err := admin.Errors(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return errorsLoadedMsg{errors: es}
	}
}

func (v *ErrorsView) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// errorRow renders one ErrorInfo as table cells. Pure: deterministic output
// for given input, multiline messages collapse to a single line.
func errorRow(e ch.ErrorInfo) []string {
	where := "local"
	if e.Remote {
		where = "remote"
	}
	msg := strings.ReplaceAll(e.LastErrorMessage, "\n", " ")
	return []string{
		e.Name,
		fmt.Sprintf("%d", e.Code),
		humanCount(e.Value),
		e.LastErrorTime.UTC().Format("2006-01-02 15:04:05"),
		where,
		msg,
	}
}

// isGrowing reports whether the current counter has strictly increased since
// the previous sample. A drop (curr < prev) means the server restarted; we
// don't flag that as growth.
func isGrowing(prev, curr uint64) bool {
	return curr > prev
}
