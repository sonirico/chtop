package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sonirico/chtop/pkg/ch"
)

// ClustersView lists every replica in every cluster the server knows about
// (system.clusters). Errors_count > 0 is flagged in the status column.
type ClustersView struct {
	app      *App
	table    table.Model
	replicas []ch.ClusterReplica
	errored  bool
}

func newClustersView(app *App) *ClustersView {
	tbl := table.New(
		table.WithColumns(clustersColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &ClustersView{app: app, table: tbl}
}

func (v *ClustersView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(clustersColumns(w))
	}
}

func clustersColumns(width int) []table.Column {
	const (
		shardW  = 6
		replW   = 6
		portW   = 6
		errW    = 7
		statusW = 8
		gutter  = 2
	)
	fixed := shardW + replW + portW + errW + statusW + gutter*6
	nameW := (width - fixed) / 2
	if nameW < 16 {
		nameW = 16
	}
	hostW := width - fixed - nameW
	if hostW < 16 {
		hostW = 16
	}
	return []table.Column{
		{Title: "CLUSTER", Width: nameW},
		{Title: "SHARD", Width: shardW},
		{Title: "REPLICA", Width: replW},
		{Title: "HOST", Width: hostW},
		{Title: "PORT", Width: portW},
		{Title: "ERRORS", Width: errW},
		{Title: "STATUS", Width: statusW},
	}
}

func (v *ClustersView) Init() tea.Cmd {
	return tea.Batch(v.load(), v.tick())
}

func (v *ClustersView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case clustersLoadedMsg:
		v.replicas = m.clusters
		v.errored = false
		v.table.SetRows(clustersToRows(m.clusters))
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewClusters || v.errored {
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

func (v *ClustersView) View() string {
	if len(v.replicas) == 0 {
		return "\n  loading clusters...\n"
	}
	return v.table.View()
}

func (v *ClustersView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cs, err := admin.Clusters(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return clustersLoadedMsg{clusters: cs}
	}
}

func (v *ClustersView) tick() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func clustersToRows(rs []ch.ClusterReplica) []table.Row {
	rows := make([]table.Row, 0, len(rs))
	for _, r := range rs {
		status := "ok"
		if r.ErrorsCount > 0 {
			status = "err"
		}
		host := r.HostName
		if host == "" {
			host = r.HostAddress
		}
		rows = append(rows, table.Row{
			r.Cluster,
			fmt.Sprintf("%d", r.ShardNum),
			fmt.Sprintf("%d", r.ReplicaNum),
			host,
			fmt.Sprintf("%d", r.Port),
			fmt.Sprintf("%d", r.ErrorsCount),
			status,
		})
	}
	return rows
}
