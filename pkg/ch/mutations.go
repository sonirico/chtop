package ch

import (
	"context"
	"fmt"
	"time"
)

// MutationInfo is one row of system.mutations for a single table. A mutation
// is an async ALTER ... UPDATE / DELETE / MATERIALIZE that ClickHouse runs in
// the background; PartsToDo > 0 with IsDone=false means it is still working.
type MutationInfo struct {
	Database         string
	Table            string
	MutationID       string
	Command          string
	CreateTime       time.Time
	BlockNumbers     uint64
	PartsToDo        uint64
	IsDone           bool
	LatestFailedPart string
	LatestFailTime   time.Time
	LatestFailReason string
}

// Mutations returns mutations for a single table. Order: in-progress first,
// then completed/failed mutations by create_time descending.
func (c *Client) Mutations(
	ctx context.Context, database, table string,
) ([]MutationInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			database, table, mutation_id, command,
			create_time,
			parts_to_do, is_done,
			latest_failed_part, latest_fail_time, latest_fail_reason
		FROM system.mutations
		WHERE database = $1 AND table = $2
		ORDER BY is_done ASC, create_time DESC
	`, database, table)
	if err != nil {
		return nil, fmt.Errorf("query system.mutations: %w", err)
	}
	defer rows.Close()

	var out []MutationInfo
	for rows.Next() {
		var m MutationInfo
		var isDone uint8
		if err := rows.Scan(
			&m.Database, &m.Table, &m.MutationID, &m.Command,
			&m.CreateTime,
			&m.PartsToDo, &isDone,
			&m.LatestFailedPart, &m.LatestFailTime, &m.LatestFailReason,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		m.IsDone = isDone == 1
		out = append(out, m)
	}
	return out, rows.Err()
}
