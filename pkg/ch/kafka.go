package ch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrKafkaNotConfigured is returned when system.kafka_consumers does not
// exist, which is what ClickHouse does on servers without the Kafka engine.
// The TUI uses this to render a friendly note instead of an error banner.
var ErrKafkaNotConfigured = errors.New("kafka engine not configured on this server")

// KafkaConsumerInfo is one row of system.kafka_consumers, narrowed to the
// columns that mean something at a glance.
type KafkaConsumerInfo struct {
	Database        string
	Table           string
	ConsumerID      string
	NumMessagesRead uint64
	NumCommits      uint64
	LastPollTime    time.Time
	LastCommitTime  time.Time
}

// KafkaConsumers returns one row per active Kafka-engine table consumer. When
// the server has no Kafka engine installed system.kafka_consumers does not
// exist and ClickHouse returns UNKNOWN_TABLE; we surface that as
// ErrKafkaNotConfigured.
func (c *Client) KafkaConsumers(ctx context.Context) ([]KafkaConsumerInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			database, table, consumer_id,
			num_messages_read, num_commits,
			last_poll_time, last_commit_time
		FROM system.kafka_consumers
		ORDER BY database, table, consumer_id
	`)
	if err != nil {
		if isUnknownTable(err) {
			return nil, ErrKafkaNotConfigured
		}
		return nil, fmt.Errorf("query system.kafka_consumers: %w", err)
	}
	defer rows.Close()

	var out []KafkaConsumerInfo
	for rows.Next() {
		var k KafkaConsumerInfo
		if err := rows.Scan(
			&k.Database, &k.Table, &k.ConsumerID,
			&k.NumMessagesRead, &k.NumCommits,
			&k.LastPollTime, &k.LastCommitTime,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// isUnknownTable reports whether err is ClickHouse's UNKNOWN_TABLE (60).
// String match against the message because clickhouse-go does not export
// typed error codes at this surface.
func isUnknownTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown_table") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "code: 60")
}
