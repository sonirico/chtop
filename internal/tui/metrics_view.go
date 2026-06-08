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

// metricsHistoryLen caps the number of samples kept per metric. The braille
// sparkline packs 2 samples per cell and stretches to fill the available
// width, so the buffer is deep enough to feed a wide column (a ~140-cell
// sparkline on a wide terminal needs ~280 samples). At the 2s tick that is
// ~10 minutes of history; the cost is a few floats per series.
const metricsHistoryLen = 300

// metricRow is one rendered line: a label, its current value, an optional
// trailing field (the per-second rate for events) and the sample history.
// fmtFn formats the window's max/avg in the same unit as value.
type metricRow struct {
	key      string
	value    string
	trailing string
	hist     []float64
	fmtFn    func(float64) string
}

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

// metricsTwoColMin is the terminal width at or above which the dashboard
// splits into two columns. Below it the boxes stack full-width.
const metricsTwoColMin = 180

func (v *MetricsView) render() {
	if !v.loaded {
		return
	}
	w := v.viewport.Width
	if w <= 0 {
		w = 80
	}

	var gauges, async, events []metricRow
	for _, k := range metricsCuratedGauges {
		cur, ok := v.current.Metrics[k]
		if !ok {
			continue
		}
		gauges = append(gauges, metricRow{
			key:   k,
			value: formatMetric(k, cur),
			hist:  v.histGauges[k],
			fmtFn: func(f float64) string { return formatMetric(k, int64(f)) },
		})
	}
	for _, k := range metricsCuratedAsync {
		fv, ok := v.current.Async[k]
		if !ok {
			continue
		}
		async = append(async, metricRow{
			key:   k,
			value: formatAsync(k, fv),
			hist:  v.histAsync[k],
			fmtFn: func(f float64) string { return formatAsync(k, f) },
		})
	}
	for _, k := range metricsCuratedEvents {
		cur, ok := v.current.Events[k]
		if !ok {
			continue
		}
		hist := v.histEvents[k]
		trailing := "(rate next tick)"
		if !v.prev.SampledAt.IsZero() && len(hist) > 0 {
			trailing = fmt.Sprintf("%.1f /s", hist[len(hist)-1])
		}
		events = append(events, metricRow{
			key:      k,
			value:    formatMetric(k, int64(cur)),
			trailing: trailing,
			hist:     hist,
			fmtFn:    func(f float64) string { return fmt.Sprintf("%.1f", f) },
		})
	}

	var out string
	if w >= metricsTwoColMin {
		colW := w / 2
		left := renderMetricBox("Current gauges", gauges, false, colW)
		right := lipgloss.JoinVertical(lipgloss.Left,
			renderMetricBox("Async / OS", async, false, w-colW),
			renderMetricBox("Events", events, true, w-colW),
		)
		out = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	} else {
		out = lipgloss.JoinVertical(lipgloss.Left,
			renderMetricBox("Current gauges", gauges, false, w),
			renderMetricBox("Async / OS", async, false, w),
			renderMetricBox("Events", events, true, w),
		)
	}
	v.viewport.SetContent(out)
}

// metricsGraphH is the height in terminal rows of the area graph drawn beneath
// each metric label.
const metricsGraphH = 3

// renderMetricBox draws one bordered section spanning outerW columns. Each
// metric renders as a one-line header (label / value / stats) followed by a
// full-width area graph, giving a btop-style look.
func renderMetricBox(title string, rows []metricRow, hasRate bool, outerW int) string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	valSt := lipgloss.NewStyle().Bold(true).Foreground(colorPrimaryHi)
	rateSt := lipgloss.NewStyle().Foreground(colorOK)
	statSt := lipgloss.NewStyle().Foreground(colorMuted)

	const keyW, valW, rateW, statsW = 24, 11, 13, 22

	innerW := outerW - 4 // rounded border (2) + horizontal padding (2)
	if innerW < 30 {
		innerW = 30
	}

	lines := make([]string, 0, len(rows)*(metricsGraphH+2)+1)
	lines = append(lines, bold.Render(title))
	for i, r := range rows {
		// Header line: label  value  [rate]  max/avg stats
		parts := []string{
			muted.Render(padRight(truncate(r.key, keyW), keyW)),
			valSt.Render(padRight(r.value, valW)),
		}
		if hasRate {
			parts = append(parts, rateSt.Render(padRight(r.trailing, rateW)))
		}
		if len(r.hist) > 0 {
			_, mx, avg := seriesStats(r.hist, innerW)
			stats := fmt.Sprintf("max %s avg %s", r.fmtFn(mx), r.fmtFn(avg))
			parts = append(parts, statSt.Render(truncate(stats, statsW)))
		}
		lines = append(lines, strings.Join(parts, "  "))

		// Area graph spanning the full inner width.
		lines = append(lines, areaGraphStyled(r.hist, innerW, metricsGraphH)...)

		if i < len(rows)-1 {
			lines = append(lines, "")
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder).
		Padding(0, 1).
		Width(innerW).
		Render(strings.Join(lines, "\n"))
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
