package ch

import (
	"context"
	"fmt"
	"time"
)

// MetricsSnapshot bundles the three sources of runtime numbers ClickHouse
// exposes: current gauge values (system.metrics), async / OS metrics
// (system.asynchronous_metrics) and cumulative event counters
// (system.events). Each is keyed by metric name.
type MetricsSnapshot struct {
	SampledAt time.Time
	Metrics   map[string]int64
	Async     map[string]float64
	Events    map[string]int64
}

// Metrics returns a single snapshot of the three system tables. Calling this
// repeatedly and diffing the Events map yields per-second rates; the view
// layer does that.
func (c *Client) Metrics(ctx context.Context) (MetricsSnapshot, error) {
	snap := MetricsSnapshot{
		SampledAt: time.Now().UTC(),
		Metrics:   map[string]int64{},
		Async:     map[string]float64{},
		Events:    map[string]int64{},
	}

	if err := c.scanInt64Pairs(ctx, &snap.Metrics,
		`SELECT metric, value FROM system.metrics`); err != nil {
		return snap, fmt.Errorf("system.metrics: %w", err)
	}
	if err := c.scanFloat64Pairs(ctx, &snap.Async,
		`SELECT metric, value FROM system.asynchronous_metrics`); err != nil {
		return snap, fmt.Errorf("system.asynchronous_metrics: %w", err)
	}
	if err := c.scanInt64Pairs(ctx, &snap.Events,
		`SELECT event, value FROM system.events`); err != nil {
		return snap, fmt.Errorf("system.events: %w", err)
	}
	return snap, nil
}

func (c *Client) scanInt64Pairs(
	ctx context.Context, dst *map[string]int64, sql string,
) error {
	rows, err := c.conn.Query(c.tagged(ctx), sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value int64
		if err := rows.Scan(&name, &value); err != nil {
			return err
		}
		(*dst)[name] = value
	}
	return rows.Err()
}

func (c *Client) scanFloat64Pairs(
	ctx context.Context, dst *map[string]float64, sql string,
) error {
	rows, err := c.conn.Query(c.tagged(ctx), sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value float64
		if err := rows.Scan(&name, &value); err != nil {
			return err
		}
		(*dst)[name] = value
	}
	return rows.Err()
}
