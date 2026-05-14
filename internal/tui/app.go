// Package tui is the bubbletea-driven terminal UI for chtop.
package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sonirico/chtop/pkg/ch"
)

type viewID int

const (
	viewTables viewID = iota
	viewProcesses
	viewClusters
	viewReplicas
	viewMerges
	viewTableDetail
	viewQueryLog
	viewExplain
	viewMetrics
	viewErrors
	viewMatViews
	viewHelp
)

// AppConfig wires the bits the App needs from the outside world.
type AppConfig struct {
	Cluster string // freeform label shown in the header (e.g. "prod-eu")
	Host    string
	Client  *ch.Client
}

// App is the root bubbletea Model.
type App struct {
	cfg    AppConfig
	client dataLoader
	keys   KeyMap
	styles Styles

	width, height int

	current viewID
	prev    viewID

	tables      *TablesView
	processes   *ProcessesView
	clusters    *ClustersView
	replicas    *ReplicasView
	merges      *MergesView
	tableDetail *TableDetailView
	queryLog    *QueryLogView
	explain     *ExplainView
	metrics     *MetricsView
	errors      *ErrorsView
	matviews    *MaterializedViewsView
	help        *HelpView

	cmdMode   bool
	cmdBuf    string
	lastErr   error
	connected bool
}

func NewApp(cfg AppConfig) (*App, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("nil client")
	}
	a := &App{
		cfg:       cfg,
		client:    cfg.Client,
		keys:      newKeyMap(),
		styles:    newStyles(),
		current:   viewTables,
		connected: true,
	}
	a.tables = newTablesView(a)
	a.processes = newProcessesView(a)
	a.clusters = newClustersView(a)
	a.replicas = newReplicasView(a)
	a.merges = newMergesView(a)
	a.tableDetail = newTableDetailView(a)
	a.queryLog = newQueryLogView(a)
	a.explain = newExplainView(a)
	a.metrics = newMetricsView(a)
	a.errors = newErrorsView(a)
	a.matviews = newMaterializedViewsView(a)
	a.help = newHelpView(a)
	return a, nil
}

func (a *App) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

func (a *App) Init() tea.Cmd {
	return a.tables.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
		a.resizeViews()
	case tea.KeyMsg:
		if cmd, handled := a.handleKey(m); handled {
			return a, cmd
		}
	case errorMsg:
		a.lastErr = m.err
		a.connected = false
	case switchViewMsg:
		return a, a.switchView(m.view)
	}

	switch a.current {
	case viewTables:
		return a, a.tables.Update(msg)
	case viewProcesses:
		return a, a.processes.Update(msg)
	case viewClusters:
		return a, a.clusters.Update(msg)
	case viewReplicas:
		return a, a.replicas.Update(msg)
	case viewMerges:
		return a, a.merges.Update(msg)
	case viewTableDetail:
		return a, a.tableDetail.Update(msg)
	case viewQueryLog:
		return a, a.queryLog.Update(msg)
	case viewExplain:
		return a, a.explain.Update(msg)
	case viewMetrics:
		return a, a.metrics.Update(msg)
	case viewErrors:
		return a, a.errors.Update(msg)
	case viewMatViews:
		return a, a.matviews.Update(msg)
	case viewHelp:
		return a, a.help.Update(msg)
	}
	return a, nil
}

func (a *App) View() string {
	body := ""
	switch a.current {
	case viewTables:
		body = a.tables.View()
	case viewProcesses:
		body = a.processes.View()
	case viewClusters:
		body = a.clusters.View()
	case viewReplicas:
		body = a.replicas.View()
	case viewMerges:
		body = a.merges.View()
	case viewTableDetail:
		body = a.tableDetail.View()
	case viewQueryLog:
		body = a.queryLog.View()
	case viewExplain:
		body = a.explain.View()
	case viewMetrics:
		body = a.metrics.View()
	case viewErrors:
		body = a.errors.View()
	case viewMatViews:
		body = a.matviews.View()
	case viewHelp:
		body = a.help.View()
	}
	if a.height > 0 {
		body = padToHeight(body, a.bodyHeight())
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		a.renderHeader(), body, a.renderFooter()) + "\x1b[0m"
}

func (a *App) bodyHeight() int {
	h := a.height - 2
	if h < 1 {
		return 1
	}
	return h
}

func (a *App) resizeViews() {
	w, h := a.width, a.bodyHeight()
	a.tables.SetSize(w, h)
	a.processes.SetSize(w, h)
	a.clusters.SetSize(w, h)
	a.replicas.SetSize(w, h)
	a.merges.SetSize(w, h)
	a.tableDetail.SetSize(w, h)
	a.queryLog.SetSize(w, h)
	a.explain.SetSize(w, h)
	a.metrics.SetSize(w, h)
	a.errors.SetSize(w, h)
	a.matviews.SetSize(w, h)
	a.help.SetSize(w, h)
}

func padToHeight(s string, h int) string {
	lines := strings.Count(s, "\n") + 1
	if lines >= h {
		return s
	}
	return s + strings.Repeat("\n", h-lines)
}

func (a *App) handleKey(k tea.KeyMsg) (tea.Cmd, bool) {
	if a.cmdMode {
		return a.handleCommandKey(k), true
	}
	switch {
	case keyMatches(k, a.keys.Quit):
		return tea.Quit, true
	case keyMatches(k, a.keys.Command):
		a.cmdMode = true
		a.cmdBuf = ""
		return nil, true
	case keyMatches(k, a.keys.Help):
		return a.toggleHelp(), true
	case keyMatches(k, a.keys.Back):
		switch a.current {
		case viewHelp:
			return a.switchView(a.prev), true
		case viewExplain:
			return a.switchView(viewQueryLog), true
		case viewProcesses,
			viewClusters,
			viewReplicas,
			viewMerges,
			viewTableDetail,
			viewQueryLog,
			viewMetrics,
			viewErrors,
			viewMatViews:
			return a.switchView(viewTables), true
		}
	}
	return nil, false
}

func (a *App) toggleHelp() tea.Cmd {
	if a.current == viewHelp {
		return a.switchView(a.prev)
	}
	return a.switchView(viewHelp)
}

func (a *App) handleCommandKey(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		a.cmdMode = false
		a.cmdBuf = ""
		return nil
	case "enter":
		cmd := a.runCommand(strings.TrimSpace(a.cmdBuf))
		a.cmdMode = false
		a.cmdBuf = ""
		return cmd
	case "backspace":
		if len(a.cmdBuf) > 0 {
			a.cmdBuf = a.cmdBuf[:len(a.cmdBuf)-1]
		}
		return nil
	}
	if len(k.String()) == 1 {
		a.cmdBuf += k.String()
	}
	return nil
}

func (a *App) runCommand(cmd string) tea.Cmd {
	switch cmd {
	case "tables", "t":
		return a.switchView(viewTables)
	case "processes", "ps", "p":
		return a.switchView(viewProcesses)
	case "clusters", "c":
		return a.switchView(viewClusters)
	case "replicas", "R":
		return a.switchView(viewReplicas)
	case "merges", "m":
		return a.switchView(viewMerges)
	case "querylog", "ql":
		return a.switchView(viewQueryLog)
	case "metrics", "met":
		return a.switchView(viewMetrics)
	case "errors", "err":
		return a.switchView(viewErrors)
	case "matviews", "mv":
		return a.switchView(viewMatViews)
	case "help", "?", "h":
		return a.toggleHelp()
	case "quit", "q":
		return tea.Quit
	}
	a.lastErr = fmt.Errorf("unknown command: %s", cmd)
	return nil
}

func (a *App) switchView(v viewID) tea.Cmd {
	if a.current != viewHelp {
		a.prev = a.current
	}
	a.current = v
	switch v {
	case viewTables:
		return a.tables.Init()
	case viewProcesses:
		return a.processes.Init()
	case viewClusters:
		return a.clusters.Init()
	case viewReplicas:
		return a.replicas.Init()
	case viewMerges:
		return a.merges.Init()
	case viewTableDetail:
		return a.tableDetail.Init()
	case viewQueryLog:
		return a.queryLog.Init()
	case viewExplain:
		return a.explain.Init()
	case viewMetrics:
		return a.metrics.Init()
	case viewErrors:
		return a.errors.Init()
	case viewMatViews:
		return a.matviews.Init()
	case viewHelp:
		return a.help.Init()
	}
	return nil
}

func (a *App) renderHeader() string {
	title := a.styles.Title.Render("chtop")
	view := a.styles.HeaderKey.Render(viewName(a.current))
	left := a.styles.Header.Render(title + "  " + view)
	right := a.styles.HeaderKey.Render(a.cfg.Cluster + "  " + a.cfg.Host)
	status := a.styles.StatusOK.Render("o connected")
	if !a.connected {
		status = a.styles.StatusErr.Render("x error")
	}
	return left + "  " + right + "  " + status
}

func (a *App) renderFooter() string {
	if a.cmdMode {
		return a.styles.CommandBar.Render(":" + a.cmdBuf)
	}
	if a.lastErr != nil {
		return a.styles.Error.Render("err: " + a.lastErr.Error())
	}
	keys := []string{
		": command", "/ filter", "? help", "r refresh", "k kill", "esc back", "q quit",
	}
	return a.styles.Footer.Render(strings.Join(keys, "  "))
}

func viewName(v viewID) string {
	switch v {
	case viewTables:
		return "tables"
	case viewProcesses:
		return "processes"
	case viewClusters:
		return "clusters"
	case viewReplicas:
		return "replicas"
	case viewMerges:
		return "merges"
	case viewTableDetail:
		return "table"
	case viewQueryLog:
		return "query log"
	case viewExplain:
		return "explain"
	case viewMetrics:
		return "metrics"
	case viewErrors:
		return "errors"
	case viewMatViews:
		return "matviews"
	case viewHelp:
		return "help"
	}
	return "?"
}

func (a *App) Run(ctx context.Context) error {
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := p.Run()
	a.Close()
	if err != nil {
		return fmt.Errorf("tui run: %w", err)
	}
	return nil
}

func keyMatches(msg tea.KeyMsg, b binding) bool {
	for _, k := range b.Keys() {
		if msg.String() == k {
			return true
		}
	}
	return false
}

type binding interface {
	Keys() []string
}
