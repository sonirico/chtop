package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sonirico/chtop/pkg/ch"
)

const explainTabCount = 4

var explainModeByTab = [explainTabCount]ch.ExplainMode{
	ch.ExplainPlan,
	ch.ExplainPipeline,
	ch.ExplainSyntax,
	ch.ExplainEstimate,
}

// ExplainView shows EXPLAIN output for one query (looked up by query_id in
// system.query_log). Four mode tabs: PLAN / PIPELINE / SYNTAX / ESTIMATE.
// Results are cached per mode so flipping tabs is instant after the first
// fetch.
type ExplainView struct {
	app      *App
	viewport viewport.Model

	queryID   string
	queryText string // displayed in the header for context
	tab       int
	cached    [explainTabCount]string
	loaded    [explainTabCount]bool
	err       error
	w, h      int
}

func newExplainView(app *App) *ExplainView {
	vp := viewport.New(80, 20)
	return &ExplainView{app: app, viewport: vp}
}

func (v *ExplainView) SetSize(w, h int) {
	v.w, v.h = w, h
	if w > 0 {
		v.viewport.Width = w
	}
	// 1 title + 1 query preview + 1 tabs + 1 rule = 4 rows.
	body := h - 4
	if body < 3 {
		body = 3
	}
	v.viewport.Height = body
	v.render()
}

// Target tells the view which query to explain. Called from query_log_view
// before switching.
func (v *ExplainView) Target(queryID, queryText string) {
	v.queryID = queryID
	v.queryText = queryText
	v.tab = 0
	v.cached = [explainTabCount]string{}
	v.loaded = [explainTabCount]bool{}
	v.err = nil
}

func (v *ExplainView) Init() tea.Cmd {
	v.render()
	return v.loadMode(v.tab)
}

func (v *ExplainView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case explainLoadedMsg:
		if m.tab >= 0 && m.tab < explainTabCount {
			v.cached[m.tab] = m.body
			v.loaded[m.tab] = true
		}
		v.render()
		return nil
	case errorMsg:
		v.err = m.err
		v.render()
		return nil
	case tea.KeyMsg:
		switch m.String() {
		case "r":
			v.loaded[v.tab] = false
			return v.loadMode(v.tab)
		case "1":
			return v.setTab(0)
		case "2":
			return v.setTab(1)
		case "3":
			return v.setTab(2)
		case "4":
			return v.setTab(3)
		case "tab", "right", "l":
			return v.setTab((v.tab + 1) % explainTabCount)
		case "shift+tab", "left", "h":
			return v.setTab((v.tab + explainTabCount - 1) % explainTabCount)
		}
	}
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return cmd
}

func (v *ExplainView) View() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	rule := lipgloss.NewStyle().Foreground(colorBorder).
		Render(strings.Repeat("-", maxInt(v.w, 1)))

	title := bold.Render("explain "+truncate(v.queryID, 36)) + "  " +
		muted.Render(oneLineQuery(truncate(v.queryText, 80)))
	return title + "\n" + v.renderTabs() + "\n" + rule + "\n" +
		v.viewport.View() + "\x1b[0m"
}

func (v *ExplainView) renderTabs() string {
	active := lipgloss.NewStyle().Bold(true).
		Foreground(colorSelFG).Background(colorPrimary).Padding(0, 1)
	inactive := lipgloss.NewStyle().Foreground(colorMuted).Padding(0, 1)
	labels := []string{"1 Plan", "2 Pipeline", "3 Syntax", "4 Estimate"}
	parts := make([]string, len(labels))
	for i, l := range labels {
		if i == v.tab {
			parts[i] = active.Render(l)
		} else {
			parts[i] = inactive.Render(l)
		}
	}
	return strings.Join(parts, "")
}

func (v *ExplainView) setTab(t int) tea.Cmd {
	if t == v.tab {
		return nil
	}
	v.tab = t
	v.render()
	if !v.loaded[t] {
		return tea.Batch(tea.ClearScreen, v.loadMode(t))
	}
	return tea.ClearScreen
}

func (v *ExplainView) render() {
	if v.queryID == "" {
		v.viewport.SetContent("\n  no query selected\n")
		return
	}
	if v.err != nil {
		v.viewport.SetContent("\n  error: " + v.err.Error() + "\n")
		return
	}
	if !v.loaded[v.tab] {
		v.viewport.SetContent("\n  loading EXPLAIN " +
			string(explainModeByTab[v.tab]) + " ...\n")
		return
	}
	body := v.cached[v.tab]
	if body == "" {
		body = "(empty result)"
	}
	v.viewport.SetContent(body)
}

func (v *ExplainView) loadMode(tab int) tea.Cmd {
	if tab < 0 || tab >= explainTabCount {
		return nil
	}
	admin := v.app.client
	queryID := v.queryID
	mode := explainModeByTab[tab]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		body, err := admin.ExplainQueryID(ctx, queryID, mode)
		if err != nil {
			return errorMsg{err: fmt.Errorf("explain %s: %w", mode, err)}
		}
		return explainLoadedMsg{tab: tab, body: body}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
