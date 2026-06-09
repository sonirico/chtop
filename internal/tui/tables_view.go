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

// TablesView is the entry-point list: every database.table the user can see,
// with engine, row count, on-disk size and part count.
type TablesView struct {
	app     *App
	table   table.Model
	tables  []ch.TableInfo
	errored bool
	filter  filter
}

func newTablesView(app *App) *TablesView {
	tbl := table.New(
		table.WithColumns(tablesColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &TablesView{app: app, table: tbl}
}

func (v *TablesView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(tablesColumns(w))
	}
}

func tablesColumns(width int) []table.Column {
	const (
		engineW = 22
		rowsW   = 10
		sizeW   = 12
		comprW  = 7
		partsW  = 7
		gutter  = 2
	)
	fixed := engineW + rowsW + sizeW + comprW + partsW + gutter*6
	nameW := width - fixed
	if nameW < 20 {
		nameW = 20
	}
	return []table.Column{
		{Title: "TABLE", Width: nameW},
		{Title: "ENGINE", Width: engineW},
		{Title: "ROWS", Width: rowsW},
		{Title: "SIZE", Width: sizeW},
		{Title: "COMPR", Width: comprW},
		{Title: "PARTS", Width: partsW},
	}
}

func (v *TablesView) Title() string {
	return fmt.Sprintf("Tables (%d)", len(v.tables))
}

func (v *TablesView) Init() tea.Cmd {
	return v.load()
}

func (v *TablesView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tablesLoadedMsg:
		v.tables = m.tables
		v.errored = false
		v.refreshRows()
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewTables || v.errored {
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
		if m.String() == "enter" {
			idx := v.table.Cursor()
			filtered := v.filtered()
			if idx >= 0 && idx < len(filtered) {
				sel := filtered[idx]
				return func() tea.Msg {
					v.app.tableDetail.Target(sel.Database, sel.Name)
					return switchViewMsg{view: viewTableDetail}
				}
			}
		}
	}
	var cmd tea.Cmd
	v.table, cmd = v.table.Update(msg)
	return cmd
}

func (v *TablesView) View() string {
	if len(v.tables) == 0 {
		return "\n  loading tables...\n"
	}
	if bar := v.filter.Bar(); bar != "" {
		return bar + "\n" + v.table.View()
	}
	return v.table.View()
}

func (v *TablesView) refreshRows() {
	rows := tablesToRows(v.filtered())
	v.table.SetRows(rows)
}

func (v *TablesView) filtered() []ch.TableInfo {
	if v.filter.query == "" {
		return v.tables
	}
	q := strings.ToLower(v.filter.query)
	out := make([]ch.TableInfo, 0, len(v.tables))
	for _, t := range v.tables {
		name := strings.ToLower(t.Database + "." + t.Name)
		if strings.Contains(name, q) {
			out = append(out, t)
		}
	}
	return out
}

func (v *TablesView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		ts, err := admin.ListTables(ctx, false)
		if err != nil {
			return errorMsg{err: err}
		}
		return tablesLoadedMsg{tables: ts}
	}
}

func (v *TablesView) tick() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func tablesToRows(ts []ch.TableInfo) []table.Row {
	rows := make([]table.Row, 0, len(ts))
	for _, t := range ts {
		ratio := "-"
		if r := t.CompressionRatio(); r > 0 {
			// Show inverse (e.g. 5.0x means data shrinks 5x on disk).
			ratio = fmt.Sprintf("%.1fx", 1/r)
		}
		rows = append(rows, table.Row{
			t.Database + "." + t.Name,
			t.Engine,
			humanCount(t.TotalRows),
			humanBytes(int64(t.TotalBytes)),
			ratio,
			fmt.Sprintf("%d", t.PartCount),
		})
	}
	return rows
}
