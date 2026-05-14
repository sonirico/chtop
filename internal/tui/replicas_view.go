package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sonirico/chtop/pkg/ch"
)

// ReplicasView surfaces replicated table health (system.replicas). Lag and
// queue size are the columns operators look at first.
type ReplicasView struct {
	app      *App
	table    table.Model
	replicas []ch.ReplicaStatus
	errored  bool
}

func newReplicasView(app *App) *ReplicasView {
	tbl := table.New(
		table.WithColumns(replicasColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &ReplicasView{app: app, table: tbl}
}

func (v *ReplicasView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(replicasColumns(w))
	}
}

func replicasColumns(width int) []table.Column {
	const (
		queueW  = 7
		mergesW = 8
		lagW    = 9
		actW    = 9
		gutter  = 2
	)
	fixed := queueW + mergesW + lagW + actW + gutter*5
	nameW := width - fixed
	if nameW < 24 {
		nameW = 24
	}
	return []table.Column{
		{Title: "TABLE", Width: nameW},
		{Title: "QUEUE", Width: queueW},
		{Title: "MERGES", Width: mergesW},
		{Title: "DELAY", Width: lagW},
		{Title: "ACTIVE", Width: actW},
	}
}

func (v *ReplicasView) Init() tea.Cmd {
	return tea.Batch(v.load(), v.tick())
}

func (v *ReplicasView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case replicasLoadedMsg:
		v.replicas = m.replicas
		v.errored = false
		v.table.SetRows(replicasToRows(m.replicas))
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewReplicas || v.errored {
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

func (v *ReplicasView) View() string {
	if len(v.replicas) == 0 {
		return "\n  no replicated tables (or still loading)\n"
	}
	return v.table.View()
}

func (v *ReplicasView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rs, err := admin.Replicas(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return replicasLoadedMsg{replicas: rs}
	}
}

func (v *ReplicasView) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func replicasToRows(rs []ch.ReplicaStatus) []table.Row {
	rows := make([]table.Row, 0, len(rs))
	for _, r := range rs {
		rows = append(rows, table.Row{
			r.Database + "." + r.Table,
			fmt.Sprintf("%d", r.QueueSize),
			fmt.Sprintf("%d", r.MergesInQueue),
			humanDuration(r.AbsoluteDelay),
			fmt.Sprintf("%d/%d", r.ActiveReplicas, r.TotalReplicas),
		})
	}
	return rows
}
