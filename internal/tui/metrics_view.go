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

// metricsHistoryLen caps the number of samples kept per metric. At the 2s
// tick that's ~1 minute of history — enough to spot trend, cheap to keep.
const metricsHistoryLen = 30

// sparkWidth is the rendered width of each metric's sparkline column.
const sparkWidth = 24

// metricsCuratedGauges is the ordered list of system.metrics keys the
// dashboard highlights.
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

// metricsCuratedEvents are cumulative counters reported as per-second rates.
var metricsCuratedEvents = []string{
	"Query",
	"SelectQuery",
	"InsertQuery",
	"InsertedRows",
	"InsertedBytes",
	"MergedRows",
	"FailedQuery",
}

// MetricsView is the live cluster dashboard with sparklines per metric.
type MetricsView struct {
	app      *App
	viewport viewport.Model
	current  ch.MetricsSnapshot
	prev     ch.MetricsSnapshot
	loaded   bool
	errored  bool

	// history holds the trailing N samples per series. Gauges store raw
	// values, async stores the float value, events store per-second rates.
	histGauges map[string][]float64
	histAsync  map[string][]float64
	histEvents map[string][]float64 // /s rates
}

func newMetricsView(app *App) *MetricsView {
	vp := viewport.New(80, 20)
	return &MetricsView{
		app:        app,
		viewport:   vp,
		histGauges: map[string][]float64{},
		histAsync:  map[string][]float64{},
		histEvents: map[string][]float64{},
	}
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
		v.pushHistory()
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

// pushHistory appends the latest sample to each per-metric ring buffer and
// trims the head when the window is full.
func (v *MetricsView) pushHistory() {
	for _, k := range metricsCuratedGauges {
		if val, ok := v.current.Metrics[k]; ok {
			v.histGauges[k] = appendCapped(v.histGauges[k], float64(val), metricsHistoryLen)
		}
	}
	for _, k := range metricsCuratedAsync {
		if val, ok := v.current.Async[k]; ok {
			v.histAsync[k] = appendCapped(v.histAsync[k], val, metricsHistoryLen)
		}
	}
	elapsed := v.current.SampledAt.Sub(v.prev.SampledAt)
	for _, k := range metricsCuratedEvents {
		if v.prev.SampledAt.IsZero() {
			continue
		}
		curr, ok := v.current.Events[k]
		if !ok {
			continue
		}
		prev := v.prev.Events[k]
		v.histEvents[k] = appendCapped(
			v.histEvents[k], perSecondRate(prev, curr, elapsed), metricsHistoryLen,
		)
	}
}

func appendCapped(buf []float64, v float64, max int) []float64 {
	buf = append(buf, v)
	if len(buf) > max {
		buf = buf[len(buf)-max:]
	}
	return buf
}

func (v *MetricsView) render() {
	if !v.loaded {
		return
	}
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	val := lipgloss.NewStyle().Bold(true).Foreground(colorPrimaryHi)
	rate := lipgloss.NewStyle().Foreground(colorOK)
	spark := lipgloss.NewStyle().Foreground(colorPrimary)

	var b strings.Builder

	b.WriteString(bold.Render("Current gauges") + "\n")
	for _, k := range metricsCuratedGauges {
		cur, ok := v.current.Metrics[k]
		if !ok {
			continue
		}
		line := metricLine(
			k, formatMetric(k, cur),
			sparkline(v.histGauges[k], sparkWidth),
			muted, val, spark,
		)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	b.WriteString(bold.Render("Async / OS") + "\n")
	for _, k := range metricsCuratedAsync {
		fv, ok := v.current.Async[k]
		if !ok {
			continue
		}
		line := metricLine(
			k, formatAsync(k, fv),
			sparkline(v.histAsync[k], sparkWidth),
			muted, val, spark,
		)
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	b.WriteString(bold.Render("Events") + "\n")
	for _, k := range metricsCuratedEvents {
		cur, ok := v.current.Events[k]
		if !ok {
			continue
		}
		history := v.histEvents[k]
		rateStr := "(rate next tick)"
		if !v.prev.SampledAt.IsZero() && len(history) > 0 {
			rateStr = fmt.Sprintf("%.1f /s", history[len(history)-1])
		}
		line := fmt.Sprintf(
			"  %s  %s  %s  %s",
			muted.Render(padRight(k, 26)),
			val.Render(padRight(formatMetric(k, int64(cur)), 12)),
			rate.Render(padRight(rateStr, 14)),
			spark.Render(sparkline(history, sparkWidth)),
		)
		b.WriteString(line + "\n")
	}

	v.viewport.SetContent(b.String())
}

// metricLine renders one row of the gauges/async sections: label, current
// value, sparkline. Pure formatting, no I/O.
func metricLine(key, value, spark string, labelStyle, valStyle, sparkStyle lipgloss.Style) string {
	return fmt.Sprintf(
		"  %s  %s  %s",
		labelStyle.Render(padRight(key, 38)),
		valStyle.Render(padRight(value, 14)),
		sparkStyle.Render(spark),
	)
}

// formatAsync renders a float async metric using a small heuristic.
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
