package ch

import (
	"context"
	"fmt"
	"time"
)

// MergeInfo is one row of system.merges: an in-flight merge or mutation.
// Progress is a 0..1 float. ResultPart names the part being produced.
type MergeInfo struct {
	Database     string
	Table        string
	Elapsed      time.Duration
	Progress     float64
	NumParts     uint64
	IsMutation   bool
	MergeType    string
	MergeAlgo    string
	RowsRead     uint64
	RowsWritten  uint64
	BytesRead    uint64
	BytesWritten uint64
	ResultPart   string
	MemoryUsage  uint64
}

// Merges returns all active merges + mutations across the server. The query
// uses tolerant column selection so it works on older ClickHouse versions
// where merge_type / merge_algorithm aren't present.
func (c *Client) Merges(ctx context.Context) ([]MergeInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			database, table, elapsed, progress, num_parts,
			is_mutation,
			toString(merge_type)      AS merge_type,
			toString(merge_algorithm) AS merge_algo,
			rows_read, rows_written,
			bytes_read_uncompressed, bytes_written_uncompressed,
			result_part_name, memory_usage
		FROM system.merges
		ORDER BY elapsed DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query system.merges: %w", err)
	}
	defer rows.Close()

	var out []MergeInfo
	for rows.Next() {
		var m MergeInfo
		var elapsedSec float64
		var isMut uint8
		if err := rows.Scan(
			&m.Database, &m.Table, &elapsedSec, &m.Progress, &m.NumParts,
			&isMut, &m.MergeType, &m.MergeAlgo,
			&m.RowsRead, &m.RowsWritten,
			&m.BytesRead, &m.BytesWritten,
			&m.ResultPart, &m.MemoryUsage,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		m.Elapsed = time.Duration(elapsedSec * float64(time.Second))
		m.IsMutation = isMut == 1
		out = append(out, m)
	}
	return out, rows.Err()
}
