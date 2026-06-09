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

// MaterializedViewsView renders each MV as a single arrow line:
//
//	src.events  ->  mv.user_counts_mv  ->  mv.user_counts  (12.3M rows, 4.5 GiB)
type MaterializedViewsView struct {
	app      *App
	viewport viewport.Model
	mvs      []ch.MaterializedViewInfo
	loaded   bool
	errored  bool
}

func newMaterializedViewsView(app *App) *MaterializedViewsView {
	vp := viewport.New(80, 20)
	return &MaterializedViewsView{app: app, viewport: vp}
}

func (v *MaterializedViewsView) SetSize(w, h int) {
	if w > 0 {
		v.viewport.Width = w
	}
	if h > 2 {
		v.viewport.Height = h
	}
	v.render()
}

func (v *MaterializedViewsView) Title() string {
	return fmt.Sprintf("Materialized views (%d)", len(v.mvs))
}

func (v *MaterializedViewsView) Init() tea.Cmd {
	return v.load()
}

func (v *MaterializedViewsView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case matviewsLoadedMsg:
		v.mvs = m.mvs
		v.loaded = true
		v.errored = false
		v.render()
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewMatViews || v.errored {
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
	v.viewport, cmd = v.viewport.Update(msg)
	return cmd
}

func (v *MaterializedViewsView) View() string {
	if !v.loaded {
		return "\n  loading materialized views...\n"
	}
	if len(v.mvs) == 0 {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("no materialized views in this server") + "\n"
	}
	return v.viewport.View()
}

func (v *MaterializedViewsView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		mvs, err := admin.MaterializedViews(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return matviewsLoadedMsg{mvs: mvs}
	}
}

func (v *MaterializedViewsView) tick() tea.Cmd {
	return tea.Tick(10*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (v *MaterializedViewsView) render() {
	if !v.loaded {
		return
	}
	v.viewport.SetContent(renderMatViews(v.mvs))
}

// renderMatViews builds the multi-line graph rendering. Pure: takes the
// snapshot and produces the styled text. Kept package-level so it can be
// covered by a snapshot test later without touching the view.
func renderMatViews(mvs []ch.MaterializedViewInfo) string {
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	src := lipgloss.NewStyle().Foreground(colorPrimaryHi)
	mv := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	tgt := lipgloss.NewStyle().Foreground(colorOK)
	arrow := muted.Render(" -> ")

	var b strings.Builder
	for _, m := range mvs {
		srcName := m.Source
		if srcName == "" {
			srcName = "?"
		}
		targetName := m.Target
		if targetName == "" {
			targetName = "(inner)"
		}
		mvName := m.Database + "." + m.Name
		stats := ""
		if m.TargetRows > 0 || m.TargetBytes > 0 {
			stats = muted.Render(fmt.Sprintf(
				"  (%s rows, %s)",
				humanCount(m.TargetRows), humanBytes(int64(m.TargetBytes)),
			))
		}
		b.WriteString(
			"  " + src.Render(srcName) + arrow +
				mv.Render(mvName) + arrow +
				tgt.Render(targetName) + stats + "\n",
		)
	}
	return b.String()
}
