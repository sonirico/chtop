package tui

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sonirico/chtop/pkg/ch"
)

// kafkaStaleAfter is how long without a poll we flag a consumer as stale.
const kafkaStaleAfter = 30 * time.Second

// KafkaConsumersView lists Kafka-engine table consumers. Servers without the
// Kafka engine show a placeholder message rather than an error banner.
type KafkaConsumersView struct {
	app           *App
	table         table.Model
	consumers     []ch.KafkaConsumerInfo
	notConfigured bool
	errored       bool
}

func newKafkaConsumersView(app *App) *KafkaConsumersView {
	tbl := table.New(
		table.WithColumns(kafkaColumns(120)),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	tbl.SetStyles(tableStyles())
	return &KafkaConsumersView{app: app, table: tbl}
}

func (v *KafkaConsumersView) SetSize(w, h int) {
	if h > 2 {
		v.table.SetHeight(h - 1)
	}
	if w > 0 {
		v.table.SetWidth(w)
		v.table.SetColumns(kafkaColumns(w))
	}
}

func kafkaColumns(width int) []table.Column {
	const (
		consumerW = 28
		readW     = 12
		commitsW  = 10
		pollW     = 14
		statusW   = 8
		gutter    = 2
	)
	fixed := consumerW + readW + commitsW + pollW + statusW + gutter*6
	tableW := width - fixed
	if tableW < 20 {
		tableW = 20
	}
	return []table.Column{
		{Title: "TABLE", Width: tableW},
		{Title: "CONSUMER", Width: consumerW},
		{Title: "READ", Width: readW},
		{Title: "COMMITS", Width: commitsW},
		{Title: "LAST POLL", Width: pollW},
		{Title: "STATUS", Width: statusW},
	}
}

func (v *KafkaConsumersView) Title() string {
	if v.notConfigured {
		return "Kafka consumers"
	}
	return fmt.Sprintf("Kafka consumers (%d)", len(v.consumers))
}

func (v *KafkaConsumersView) Init() tea.Cmd {
	return v.load()
}

func (v *KafkaConsumersView) Update(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case kafkaConsumersLoadedMsg:
		v.consumers = m.consumers
		v.errored = false
		v.notConfigured = false
		v.refreshRows()
		return v.tick()
	case kafkaNotConfiguredMsg:
		v.notConfigured = true
		v.consumers = nil
		v.errored = false
		v.table.SetRows(nil)
		return v.tick()
	case errorMsg:
		v.errored = true
		return nil
	case tickMsg:
		if v.app.current != viewKafka || v.errored {
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
	v.table, cmd = v.table.Update(msg)
	return cmd
}

func (v *KafkaConsumersView) View() string {
	if v.notConfigured {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("Kafka engine not configured on this server.") + "\n"
	}
	if !v.errored && len(v.consumers) == 0 {
		muted := lipgloss.NewStyle().Foreground(colorMuted)
		return "\n  " + muted.Render("no active Kafka-engine consumers") + "\n"
	}
	return v.table.View()
}

func (v *KafkaConsumersView) refreshRows() {
	now := time.Now().UTC()
	rows := make([]table.Row, 0, len(v.consumers))
	hot := lipgloss.NewStyle().Foreground(colorErr).Bold(true)
	for _, k := range v.consumers {
		row := kafkaConsumerRow(now, k)
		if row[5] == "stale" {
			row[5] = hot.Render("stale")
		}
		rows = append(rows, table.Row(row))
	}
	v.table.SetRows(rows)
}

func (v *KafkaConsumersView) load() tea.Cmd {
	admin := v.app.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cs, err := admin.KafkaConsumers(ctx)
		if err != nil {
			if errors.Is(err, ch.ErrKafkaNotConfigured) {
				return kafkaNotConfiguredMsg{}
			}
			return errorMsg{err: err}
		}
		return kafkaConsumersLoadedMsg{consumers: cs}
	}
}

func (v *KafkaConsumersView) tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// kafkaConsumerRow renders a row of cells for the consumers table. Pure:
// takes `now` as a parameter so the staleness flag is deterministic in tests.
func kafkaConsumerRow(now time.Time, k ch.KafkaConsumerInfo) []string {
	pollCell := "never"
	status := "stale"
	if !k.LastPollTime.IsZero() {
		age := now.Sub(k.LastPollTime)
		pollCell = humanDuration(age) + " ago"
		if age < kafkaStaleAfter {
			status = "ok"
		}
	}
	return []string{
		fmt.Sprintf("%s.%s", k.Database, k.Table),
		k.ConsumerID,
		humanCount(k.NumMessagesRead),
		humanCount(k.NumCommits),
		pollCell,
		status,
	}
}
