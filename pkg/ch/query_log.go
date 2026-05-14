package ch

import (
	"context"
	"fmt"
	"time"
)

// Query type codes used by system.query_log. ClickHouse stores `type` as
// Enum8 with these values; we expose them as named constants so chtop and
// other callers don't sprinkle magic numbers around.
const (
	QueryTypeStart                    uint8 = 1
	QueryTypeFinish                   uint8 = 2
	QueryTypeExceptionBeforeStart     uint8 = 3
	QueryTypeExceptionWhileProcessing uint8 = 4
)

// QueryLogInfo is one row of system.query_log filtered to "terminating"
// event types (Finish / Exception*). Used by the query log tail.
type QueryLogInfo struct {
	EventTime   time.Time
	DurationMs  uint64
	ReadRows    uint64
	ReadBytes   uint64
	MemoryUsage uint64
	Type        uint8
	Query       string
	User        string
	Database    string
	QueryID     string
	Exception   string
}

// IsError reports whether this entry represents a failed query (either an
// exception before start or while processing).
func (q QueryLogInfo) IsError() bool {
	return q.Type == QueryTypeExceptionBeforeStart ||
		q.Type == QueryTypeExceptionWhileProcessing
}

// QueryLog returns terminating events from system.query_log newer than
// `since`, newest first. chtop's own queries (tagged with QueryIDPrefix) are
// excluded.
func (c *Client) QueryLog(
	ctx context.Context, since time.Time, limit int,
) ([]QueryLogInfo, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			event_time,
			query_duration_ms,
			read_rows, read_bytes, memory_usage,
			toUInt8(type) AS type_code,
			query, user, current_database, query_id, exception
		FROM system.query_log
		WHERE type IN (2, 3, 4)
		  AND query_id NOT LIKE concat($1, '%')
		  AND event_time >= $2
		ORDER BY event_time DESC
		LIMIT $3
	`, QueryIDPrefix, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query system.query_log: %w", err)
	}
	defer rows.Close()

	var out []QueryLogInfo
	for rows.Next() {
		var q QueryLogInfo
		if err := rows.Scan(
			&q.EventTime,
			&q.DurationMs,
			&q.ReadRows, &q.ReadBytes, &q.MemoryUsage,
			&q.Type,
			&q.Query, &q.User, &q.Database, &q.QueryID, &q.Exception,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}
