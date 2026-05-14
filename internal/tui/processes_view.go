package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sonirico/chtop/pkg/ch"
)

// ProcessesView lists running queries (system.processes) and lets the user
// kill them with `k`. This is the killer feature of chtop: a DBA opens it,
// spots the runaway query, hits k, done.
type ProcessesView struct {
	app     *App
	table   table.Model
	procs   []ch.Process
	errored bool

	confirming bool   // when true, `k` was pressed; wait for confirmation
	pendingID  string // query id awaiting kill confirmation
}

func newProcessesView(app *App) *ProcessesView {
	tbl := table.New(
		table.WithColumns(processesColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &ProcessesView{app: app, table: tbl}
}

func (v *ProcessesView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 2)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(processesColumns(w))
	}
}

func processesColumns(width int) []table.Column {
	const (
		userW    = 16
		elapsedW = 9
		memW     = 10
		rowsW    = 10
		gutter   = 2
	)
	fixed := userW + elapsedW + memW + rowsW + gutter*5
	queryW := width - fixed
	if queryW < 30 {
		queryW = 30
	}
	return []table.Column{
		{Title: "USER", Width: userW},
		{Title: "ELAPSED", Width: elapsedW},
		{Title: "MEMORY", Width: memW},
		{Title: "READ ROWS", Width: rowsW},
		{Title: "QUERY", Width: queryW},
	}
}

func (v *ProcessesView) Init() tea.Cmd {
	return v.load()
}

func (v *ProcessesView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case processesLoadedMsg:
		v.procs = m.procs
		v.errored = false
		v.table.SetRows(processesToRows(m.procs))
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case killDoneMsg:
		v.confirming = false
		v.pendingID = ""
		return v.load()
	case tickMsg:
		if v.app.current != viewProcesses || v.errored {
			return nil
		}
		return v.load()
	case tea.KeyMsg:
		if v.confirming {
			switch m.String() {
			case "y", "Y":
				cmd := v.kill(v.pendingID)
				v.confirming = false
				v.pendingID = ""
				return cmd
			default:
				v.confirming = false
				v.pendingID = ""
				return nil
			}
		}
		switch m.String() {
		case "r":
			v.errored = false
			return v.load()
		case "k":
			if p, ok := v.selected(); ok {
				v.confirming = true
				v.pendingID = p.QueryID
				return nil
			}
		}
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return cmd
}

func (v *ProcessesView) View() string {
	if v.confirming {
		short := v.pendingID
		if len(short) > 36 {
			short = short[:36]
		}
		return v.table.View() + "\n  kill query " + short + "? (y/N)"
	}
	if len(v.procs) == 0 {
		return "\n  no running queries\n"
	}
	return v.table.View()
}

func (v *ProcessesView) selected() (ch.Process, bool) {
	idx := v.table.Cursor()
	if idx < 0 || idx >= len(v.procs) {
		return ch.Process{}, false
	}
	return v.procs[idx], true
}

func (v *ProcessesView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ps, err := admin.Processes(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return processesLoadedMsg{procs: ps}
	}
}

func (v *ProcessesView) kill(queryID string) tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := admin.KillQuery(ctx, queryID)
		return killDoneMsg{queryID: queryID, err: err}
	}
}

func (v *ProcessesView) tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func processesToRows(ps []ch.Process) []table.Row {
	rows := make([]table.Row, 0, len(ps))
	for _, p := range ps {
		query := strings.ReplaceAll(p.Query, "\n", " ")
		query = strings.TrimSpace(query)
		rows = append(rows, table.Row{
			p.User,
			humanDuration(p.Elapsed),
			humanBytes(p.PeakMemoryUsage),
			humanCount(p.ReadRows),
			query,
		})
	}
	return rows
}
