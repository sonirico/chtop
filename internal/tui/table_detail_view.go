// Package tui — table detail view.
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

type detailTab int

const (
	tabSchema detailTab = iota
	tabParts
	tabMutations
	tabEngine
)

const detailTabCount = 4

// TableDetailView is the per-table drill-down: schema, parts, mutations,
// engine info. Five sections of the kind a DBA wants in one screen.
type TableDetailView struct {
	app      *App
	viewport viewport.Model

	database string
	name     string

	describe       ch.TableDescription
	describeLoaded bool

	columns       []ch.ColumnInfo
	columnsLoaded bool

	parts       []ch.PartInfo
	partsLoaded bool

	mutations       []ch.MutationInfo
	mutationsLoaded bool

	err error

	tab     detailTab
	offsets [detailTabCount]int
	w, h    int
}

func newTableDetailView(app *App) *TableDetailView {
	vp := viewport.New(80, 20)
	return &TableDetailView{app: app, viewport: vp}
}

func (v *TableDetailView) Title() string {
	if v.name == "" {
		return "Table"
	}
	return "Table  " + v.database + "." + v.name
}

func (v *TableDetailView) SetSize(w, h int) {
	v.w, v.h = w, h
	if w > 0 {
		v.viewport.Width = w
	}
	// 1 tabs + 1 rule = 2 rows of chrome (the title is in the panel border).
	body := h - 2
	if body < 3 {
		body = 3
	}
	v.viewport.Height = body
	v.refresh()
}

// Target tells the view which table to show. Called from tables_view before
// switching.
func (v *TableDetailView) Target(database, table string) {
	v.database = database
	v.name = table
}

func (v *TableDetailView) Init() tea.Cmd {
	v.describe = ch.TableDescription{}
	v.describeLoaded = false
	v.columns = nil
	v.columnsLoaded = false
	v.parts = nil
	v.partsLoaded = false
	v.mutations = nil
	v.mutationsLoaded = false
	v.err = nil
	v.tab = tabSchema
	v.offsets = [detailTabCount]int{}
	v.refresh()
	return tea.Batch(v.loadDescribe(), v.loadColumns(), v.loadParts(), v.loadMutations())
}

func (v *TableDetailView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case tableDescribeLoadedMsg:
		v.describe = m.desc
		v.describeLoaded = true
		v.err = nil
		v.refresh()
		return nil
	case columnsLoadedMsg:
		v.columns = m.columns
		v.columnsLoaded = true
		v.refresh()
		return nil
	case partsLoadedMsg:
		v.parts = m.parts
		v.partsLoaded = true
		v.refresh()
		return nil
	case tableMutationsLoadedMsg:
		v.mutations = m.mutations
		v.mutationsLoaded = true
		v.refresh()
		return nil
	case errorMsg:
		v.err = m.err
		v.refresh()
		return nil
	case tea.KeyMsg:
		switch m.String() {
		case "r":
			return v.Init()
		case "1":
			return v.setTab(tabSchema)
		case "2":
			return v.setTab(tabParts)
		case "3":
			return v.setTab(tabMutations)
		case "4":
			return v.setTab(tabEngine)
		case "tab", "right", "l":
			return v.setTab(detailTab((int(v.tab) + 1) % detailTabCount))
		case "shift+tab", "left", "h":
			return v.setTab(detailTab((int(v.tab) + detailTabCount - 1) % detailTabCount))
		}
	}
	var cmd tea.Cmd
	v.viewport, cmd = v.viewport.Update(msg)
	return cmd
}

func (v *TableDetailView) View() string {
	tabs := renderTabBar(int(v.tab),
		[]string{"1 Schema", "2 Parts", "3 Mutations", "4 Engine"})
	return tabs + "\n" + renderRule(maxInt(v.w, 1)) + "\n" + v.viewport.View() + "\x1b[0m"
}

func (v *TableDetailView) setTab(t detailTab) tea.Cmd {
	if t == v.tab {
		return nil
	}
	v.offsets[v.tab] = v.viewport.YOffset
	v.tab = t
	v.refresh()
	v.viewport.SetYOffset(v.offsets[v.tab])
	return tea.ClearScreen
}

func (v *TableDetailView) refresh() {
	if v.name == "" {
		v.viewport.SetContent("\n  no table selected\n")
		return
	}
	if v.err != nil {
		v.viewport.SetContent("\n  error: " + v.err.Error() + "\n")
		return
	}
	switch v.tab {
	case tabSchema:
		v.viewport.SetContent(v.renderSchema())
	case tabParts:
		v.viewport.SetContent(v.renderParts())
	case tabMutations:
		v.viewport.SetContent(v.renderMutations())
	case tabEngine:
		v.viewport.SetContent(v.renderEngine())
	}
}

func (v *TableDetailView) renderSchema() string {
	if !v.columnsLoaded {
		return "\n  loading columns...\n"
	}
	if len(v.columns) == 0 {
		return "\n  no columns\n"
	}
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	var b strings.Builder
	header := fmt.Sprintf(
		"  %-30s %-30s %-25s %-10s %-7s %s",
		"NAME", "TYPE", "CODEC", "SIZE", "COMPR", "KEYS",
	)
	b.WriteString(muted.Render(header) + "\n")
	for _, c := range v.columns {
		row := columnRow(c)
		b.WriteString(fmt.Sprintf(
			"  %-30s %-30s %-25s %-10s %-7s %s\n",
			row[0], row[1], row[2], row[3], row[4], row[5],
		))
	}
	return b.String()
}

func (v *TableDetailView) renderParts() string {
	if !v.partsLoaded {
		return "\n  loading parts...\n"
	}
	if len(v.parts) == 0 {
		return "\n  no active parts\n"
	}
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	var b strings.Builder
	header := fmt.Sprintf(
		"  %-32s %-14s %-12s %-12s %-7s %-5s %s",
		"NAME", "PARTITION", "ROWS", "SIZE", "COMPR", "LVL", "MODIFIED",
	)
	b.WriteString(muted.Render(header) + "\n")
	for _, p := range v.parts {
		row := partRow(p)
		b.WriteString(fmt.Sprintf(
			"  %-32s %-14s %-12s %-12s %-7s %-5s %s\n",
			row[0], row[1], row[2], row[3], row[4], row[5], row[6],
		))
	}
	return b.String()
}

func (v *TableDetailView) renderMutations() string {
	if !v.mutationsLoaded {
		return "\n  loading mutations...\n"
	}
	if len(v.mutations) == 0 {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("no mutations on this table") + "\n"
	}
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	var b strings.Builder
	header := fmt.Sprintf(
		"  %-12s %-50s %-9s %-20s %-8s %s",
		"ID", "COMMAND", "STATUS", "CREATED", "PARTS", "FAIL",
	)
	b.WriteString(muted.Render(header) + "\n")
	for _, m := range v.mutations {
		row := mutationRow(m)
		cmd := row[1]
		if len(cmd) > 50 {
			cmd = cmd[:47] + "..."
		}
		fail := row[5]
		if len(fail) > 60 {
			fail = fail[:57] + "..."
		}
		b.WriteString(fmt.Sprintf(
			"  %-12s %-50s %-9s %-20s %-8s %s\n",
			row[0], cmd, row[2], row[3], row[4], fail,
		))
	}
	return b.String()
}

func (v *TableDetailView) renderEngine() string {
	if !v.describeLoaded {
		return "\n  loading engine info...\n"
	}
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	bold := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	d := v.describe

	row := func(k, val string) string {
		if val == "" {
			val = "-"
		}
		return "  " + muted.Render(padRight(k+":", 18)) + val + "\n"
	}
	var b strings.Builder
	b.WriteString(row("Engine", d.Engine))
	b.WriteString(row("Engine full", d.EngineFull))
	b.WriteString(row("Partition key", d.PartitionKey))
	b.WriteString(row("Sorting key", d.SortingKey))
	b.WriteString(row("Primary key", d.PrimaryKey))
	b.WriteString(row("Sampling key", d.SamplingKey))
	b.WriteString(row("Storage policy", d.StoragePolicy))
	b.WriteString(row("Comment", d.Comment))
	if d.CreateTableQuery != "" {
		b.WriteString("\n" + bold.Render("  CREATE TABLE") + "\n\n")
		for _, line := range strings.Split(d.CreateTableQuery, "\n") {
			b.WriteString("    " + line + "\n")
		}
	}
	return b.String()
}

func (v *TableDetailView) loadDescribe() tea.Cmd {
	admin := v.app.client
	db, t := v.database, v.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		d, err := admin.DescribeTable(ctx, db, t)
		if err != nil {
			return errorMsg{err: fmt.Errorf("describe %s.%s: %w", db, t, err)}
		}
		return tableDescribeLoadedMsg{desc: d}
	}
}

func (v *TableDetailView) loadColumns() tea.Cmd {
	admin := v.app.client
	db, t := v.database, v.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cs, err := admin.Columns(ctx, db, t)
		if err != nil {
			return errorMsg{err: fmt.Errorf("columns %s.%s: %w", db, t, err)}
		}
		return columnsLoadedMsg{columns: cs}
	}
}

func (v *TableDetailView) loadParts() tea.Cmd {
	admin := v.app.client
	db, t := v.database, v.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ps, err := admin.Parts(ctx, db, t)
		if err != nil {
			return errorMsg{err: fmt.Errorf("parts %s.%s: %w", db, t, err)}
		}
		return partsLoadedMsg{parts: ps}
	}
}

func (v *TableDetailView) loadMutations() tea.Cmd {
	admin := v.app.client
	db, t := v.database, v.name
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		ms, err := admin.Mutations(ctx, db, t)
		if err != nil {
			return errorMsg{err: fmt.Errorf("mutations %s.%s: %w", db, t, err)}
		}
		return tableMutationsLoadedMsg{mutations: ms}
	}
}

// columnRow renders a column's row cells for the schema tab. Pure: same
// input yields the same output, no clock / RNG dependencies.
func columnRow(c ch.ColumnInfo) []string {
	codec := c.CompressionCodec
	if codec == "" {
		codec = "default"
	}
	ratio := "-"
	if r := c.CompressionRatio(); r > 0 {
		ratio = fmt.Sprintf("%.1fx", r)
	}
	keys := []string{}
	if c.IsInPrimaryKey {
		keys = append(keys, "PK")
	}
	if c.IsInPartitionKey {
		keys = append(keys, "PART")
	}
	if c.IsInSortingKey {
		keys = append(keys, "SORT")
	}
	return []string{
		c.Name,
		c.Type,
		codec,
		humanBytes(int64(c.DataCompressed)),
		ratio,
		strings.Join(keys, " "),
	}
}

// mutationRow renders a mutation row for the Mutations tab. Pure: the command
// has its newlines collapsed to spaces so it fits on one line. The fail
// column is empty when there's no failure recorded yet.
func mutationRow(m ch.MutationInfo) []string {
	status := "running"
	if m.IsDone {
		status = "done"
	}
	cmd := strings.ReplaceAll(m.Command, "\n", " ")
	return []string{
		m.MutationID,
		cmd,
		status,
		m.CreateTime.UTC().Format("2006-01-02 15:04:05"),
		fmt.Sprintf("%d", m.PartsToDo),
		m.LatestFailReason,
	}
}

// partRow renders a part's row cells for the parts tab. Pure.
func partRow(p ch.PartInfo) []string {
	ratio := "-"
	if r := p.CompressionRatio(); r > 0 {
		ratio = fmt.Sprintf("%.1fx", r)
	}
	mod := p.ModificationTime.UTC().Format("2006-01-02 15:04:05")
	return []string{
		p.Name,
		p.Partition,
		humanCount(p.Rows),
		humanBytes(int64(p.BytesOnDisk)),
		ratio,
		fmt.Sprintf("%d", p.Level),
		mod,
	}
}
