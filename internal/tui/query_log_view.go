package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sonirico/chtop/pkg/ch"
)

// QueryLogView shows the most recent terminating events from
// system.query_log. Filter (`/`) matches against the query text.
type QueryLogView struct {
	app     *App
	table   table.Model
	entries []ch.QueryLogInfo
	filter  filter
	errored bool
}

func newQueryLogView(app *App) *QueryLogView {
	tbl := table.New(
		table.WithColumns(queryLogColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &QueryLogView{app: app, table: tbl}
}

func (v *QueryLogView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(queryLogColumns(w))
	}
}

func queryLogColumns(width int) []table.Column {
	const (
		timeW   = 8
		userW   = 14
		dbW     = 14
		durW    = 8
		memW    = 10
		rowsW   = 9
		statusW = 6
		gutter  = 2
	)
	fixed := timeW + userW + dbW + durW + memW + rowsW + statusW + gutter*8
	queryW := width - fixed
	if queryW < 30 {
		queryW = 30
	}
	return []table.Column{
		{Title: "TIME", Width: timeW},
		{Title: "USER", Width: userW},
		{Title: "DB", Width: dbW},
		{Title: "DURATION", Width: durW},
		{Title: "MEMORY", Width: memW},
		{Title: "ROWS", Width: rowsW},
		{Title: "STATUS", Width: statusW},
		{Title: "QUERY", Width: queryW},
	}
}

func (v *QueryLogView) Title() string {
	return fmt.Sprintf("Query log (%d)", len(v.entries))
}

func (v *QueryLogView) Init() tea.Cmd {
	return v.load()
}

func (v *QueryLogView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case queryLogLoadedMsg:
		v.entries = m.entries
		v.errored = false
		v.refreshRows()
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewQueryLog || v.errored {
			return nil
		}
		return v.load()
	case tea.KeyMsg:
		if consumed, applied := v.filter.Handle(m); consumed {
			if applied {
				v.refreshRows()
			}
			return nil
		}
		if m.String() == "r" {
			v.errored = false
			return v.load()
		}
		if m.String() == "e" || m.String() == "enter" {
			filtered := v.filtered()
			idx := v.table.Cursor()
			if idx >= 0 && idx < len(filtered) {
				sel := filtered[idx]
				return func() tea.Msg {
					v.app.explain.Target(sel.QueryID, sel.Query)
					return switchViewMsg{view: viewExplain}
				}
			}
		}
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return cmd
}

func (v *QueryLogView) View() string {
	if !v.errored && len(v.entries) == 0 {
		return "\n  loading recent queries...\n"
	}
	if bar := v.filter.Bar(); bar != "" {
		return bar + "\n" + v.table.View()
	}
	return v.table.View()
}

func (v *QueryLogView) refreshRows() {
	v.table.SetRows(queryLogToRows(v.filtered()))
}

func (v *QueryLogView) filtered() []ch.QueryLogInfo {
	if v.filter.query == "" {
		return v.entries
	}
	q := strings.ToLower(v.filter.query)
	out := make([]ch.QueryLogInfo, 0, len(v.entries))
	for _, e := range v.entries {
		if strings.Contains(strings.ToLower(e.Query), q) ||
			strings.Contains(strings.ToLower(e.User), q) ||
			strings.Contains(strings.ToLower(e.Database), q) {
			out = append(out, e)
		}
	}
	return out
}

func (v *QueryLogView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		// Pull the last 10 minutes; 200 rows max keeps the UI snappy.
		since := time.Now().UTC().Add(-10 * time.Minute)
		es, err := admin.QueryLog(ctx, since, 200)
		if err != nil {
			return errorMsg{err: err}
		}
		return queryLogLoadedMsg{entries: es}
	}
}

func (v *QueryLogView) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func queryLogToRows(es []ch.QueryLogInfo) []table.Row {
	rows := make([]table.Row, 0, len(es))
	for _, e := range es {
		rows = append(rows, table.Row(queryLogRow(e)))
	}
	return rows
}

// queryLogRow renders one query_log entry as a row of strings for the table.
// Pure: same inputs yield the same outputs; multiline queries are collapsed
// to one line so the table stays single-row-per-query.
func queryLogRow(q ch.QueryLogInfo) []string {
	status := "ok"
	if q.IsError() {
		status = "err"
	}
	return []string{
		q.EventTime.UTC().Format("15:04:05"),
		q.User,
		q.Database,
		humanDuration(time.Duration(q.DurationMs) * time.Millisecond),
		humanBytes(int64(q.MemoryUsage)),
		humanCount(q.ReadRows),
		status,
		oneLineQuery(q.Query),
	}
}

func oneLineQuery(s string) string {
	out := strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(out, "  ") {
		out = strings.ReplaceAll(out, "  ", " ")
	}
	return strings.TrimSpace(out)
}
