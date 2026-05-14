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

// MergesView shows ongoing merges and mutations as a coloured progress bar
// per row. Refresh is fast (1 s) because merges complete in seconds.
type MergesView struct {
	app     *App
	table   table.Model
	merges  []ch.MergeInfo
	errored bool
}

func newMergesView(app *App) *MergesView {
	tbl := table.New(
		table.WithColumns(mergesColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &MergesView{app: app, table: tbl}
}

func (v *MergesView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(mergesColumns(w))
	}
}

func mergesColumns(width int) []table.Column {
	const (
		barW     = 30
		pctW     = 6
		elapsedW = 8
		typeW    = 9
		rateW    = 11
		ratioW   = 7
		gutter   = 2
	)
	fixed := barW + pctW + elapsedW + typeW + rateW + ratioW + gutter*7
	nameW := width - fixed
	if nameW < 20 {
		nameW = 20
	}
	return []table.Column{
		{Title: "TABLE", Width: nameW},
		{Title: "PROGRESS", Width: barW},
		{Title: "%", Width: pctW},
		{Title: "ELAPSED", Width: elapsedW},
		{Title: "TYPE", Width: typeW},
		{Title: "READ/s", Width: rateW},
		{Title: "COMPR", Width: ratioW},
	}
}

func (v *MergesView) Init() tea.Cmd {
	return tea.Batch(v.load(), v.tick())
}

func (v *MergesView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case mergesLoadedMsg:
		v.merges = m.merges
		v.errored = false
		v.table.SetRows(v.toRows())
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewMerges || v.errored {
			return nil
		}
		return tea.Batch(v.load(), v.tick())
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

func (v *MergesView) View() string {
	if len(v.merges) == 0 {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("no merges or mutations running") + "\n"
	}
	return v.table.View()
}

func (v *MergesView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ms, err := admin.Merges(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return mergesLoadedMsg{merges: ms}
	}
}

func (v *MergesView) tick() tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (v *MergesView) toRows() []table.Row {
	rows := make([]table.Row, 0, len(v.merges))
	for _, m := range v.merges {
		bar := renderBar(m.Progress, 30, progressColor(m.Progress), colorBorder)
		pct := fmt.Sprintf("%3.0f%%", m.Progress*100)

		typ := m.MergeAlgo
		if m.IsMutation {
			typ = "MUTATION"
		} else if typ == "" {
			typ = m.MergeType
		}

		read := "-"
		if m.Elapsed > 0 && m.BytesRead > 0 {
			perSec := float64(m.BytesRead) / m.Elapsed.Seconds()
			read = humanBytes(int64(perSec)) + "/s"
		}

		ratio := "-"
		if m.BytesRead > 0 && m.BytesWritten > 0 {
			ratio = fmt.Sprintf("%.2fx", float64(m.BytesRead)/float64(m.BytesWritten))
		}

		rows = append(rows, table.Row{
			m.Database + "." + m.Table,
			bar,
			pct,
			humanDuration(m.Elapsed),
			strings.ToLower(typ),
			read,
			ratio,
		})
	}
	return rows
}
