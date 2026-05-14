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

// metricsCuratedGauges is the ordered list of system.metrics keys the
// dashboard highlights. They cover the "is the cluster healthy right now"
// question without flooding the screen with the ~150 raw metrics.
var metricsCuratedGauges = []string{
	"Query",
	"Merge",
	"PartMutation",
	"BackgroundMergesAndMutationsPoolTask",
	"BackgroundFetchesPoolTask",
	"ReplicatedFetch",
	"ReplicatedSend",
	"DistributedSend",
	"TCPConnection",
	"HTTPConnection",
	"MemoryTracking",
	"MMappedFiles",
}

// metricsCuratedAsync are the async metrics worth pinning.
var metricsCuratedAsync = []string{
	"MemoryResident",
	"Uptime",
	"LoadAverage1",
	"MaxPartCountForPartition",
	"ReplicasMaxAbsoluteDelay",
	"ReplicasMaxQueueSize",
	"jemalloc.allocated",
}

// metricsCuratedEvents are the cumulative counters the dashboard reports as
// per-second rates. Each appears with both its absolute value and a /s rate
// computed against the previous snapshot.
var metricsCuratedEvents = []string{
	"Query",
	"SelectQuery",
	"InsertQuery",
	"InsertedRows",
	"InsertedBytes",
	"MergedRows",
	"FailedQuery",
}

// MetricsView is the live cluster dashboard.
type MetricsView struct {
	app      *App
	viewport viewport.Model
	current  ch.MetricsSnapshot
	prev     ch.MetricsSnapshot
	loaded   bool
	errored  bool
}

func newMetricsView(app *App) *MetricsView {
	vp := viewport.New(80, 20)
	return &MetricsView{app: app, viewport: vp}
}

func (v *MetricsView) SetSize(w, h int) {
	if w > 0 {
		v.viewport.Width = w
	}
	if h > 2 {
		v.viewport.Height = h
	}
	v.render()
}

func (v *MetricsView) Init() tea.Cmd {
	return v.load()
}

func (v *MetricsView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case metricsLoadedMsg:
		v.prev = v.current
		v.current = m.snapshot
		v.loaded = true
		v.errored = false
		v.render()
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewMetrics || v.errored {
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

func (v *MetricsView) View() string {
	if !v.loaded {
		return "\n  loading metrics...\n"
	}
	return v.viewport.View()
}

func (v *MetricsView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		snap, err := admin.Metrics(ctx)
		if err != nil {
			return errorMsg{err: err}
		}
		return metricsLoadedMsg{snapshot: snap}
	}
}

func (v *MetricsView) tick() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (v *MetricsView) render() {
	if !v.loaded {
		return
	}
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	val := lipgloss.NewStyle().Bold(true).Foreground(colorPrimaryHi)
	rate := lipgloss.NewStyle().Foreground(colorOK)

	var b strings.Builder

	b.WriteString(bold.Render("Current gauges") + "\n")
	b.WriteString(renderGaugeColumns(v.current.Metrics, metricsCuratedGauges, val, muted))
	b.WriteString("\n")

	b.WriteString(bold.Render("Async / OS") + "\n")
	for _, k := range metricsCuratedAsync {
		fv, ok := v.current.Async[k]
		if !ok {
			continue
		}
		b.WriteString(fmt.Sprintf(
			"  %s  %s\n",
			muted.Render(padRight(k, 32)),
			val.Render(formatAsync(k, fv)),
		))
	}
	b.WriteString("\n")

	b.WriteString(bold.Render("Events") + "\n")
	elapsed := v.current.SampledAt.Sub(v.prev.SampledAt)
	for _, k := range metricsCuratedEvents {
		curr, ok := v.current.Events[k]
		if !ok {
			continue
		}
		prev := v.prev.Events[k]
		r := perSecondRate(prev, curr, elapsed)
		line := fmt.Sprintf(
			"  %s  %s",
			muted.Render(padRight(k, 24)),
			val.Render(padRight(formatMetric(k, curr), 14)),
		)
		if v.prev.SampledAt.IsZero() {
			line += "  " + muted.Render("(rate next tick)")
		} else {
			line += "  " + rate.Render(fmt.Sprintf("%.1f /s", r))
		}
		b.WriteString(line + "\n")
	}

	v.viewport.SetContent(b.String())
}

// renderGaugeColumns lays out current-value gauges in two columns to take
// advantage of wide terminals. Pure-ish: depends on a Style for rendering
// but the layout decision is data-driven.
func renderGaugeColumns(
	src map[string]int64, keys []string, val, muted lipgloss.Style,
) string {
	left := []string{}
	right := []string{}
	for i, k := range keys {
		v, ok := src[k]
		if !ok {
			continue
		}
		line := fmt.Sprintf(
			"  %s  %s",
			muted.Render(padRight(k, 38)),
			val.Render(formatMetric(k, v)),
		)
		if i%2 == 0 {
			left = append(left, line)
		} else {
			right = append(right, line)
		}
	}
	var b strings.Builder
	for i := 0; i < len(left) || i < len(right); i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		b.WriteString(padRight(l, 60) + r + "\n")
	}
	return b.String()
}

// formatAsync renders a float async metric using a small heuristic: bytes if
// the metric is in the bytes table, integer for counts, two decimals
// otherwise.
func formatAsync(key string, value float64) string {
	switch metricUnits[key] {
	case unitBytes:
		return humanBytes(int64(value))
	case unitDuration:
		return humanDuration(time.Duration(value) * time.Second)
	}
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return fmt.Sprintf("%.2f", value)
}
