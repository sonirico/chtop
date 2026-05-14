package ch

import (
	"context"
	"fmt"
	"time"
)

// PartInfo is one row of system.parts for a single table. Only the columns
// the table detail view consumes are scanned; ClickHouse exposes ~50.
type PartInfo struct {
	Name             string
	Active           bool
	Partition        string
	Rows             uint64
	BytesOnDisk      uint64
	DataCompressed   uint64
	DataUncompressed uint64
	MarksBytes       uint64
	ModificationTime time.Time
	RemovalTime      time.Time
	Level            uint32
	DataVersion      uint64
	MinDate          time.Time
	MaxDate          time.Time
	Engine           string
	DiskName         string
	Path             string
}

// Parts returns every active part of a single table, newest first. Inactive
// parts (recently merged away, awaiting cleanup) are dropped — the detail
// view only cares about live data.
func (c *Client) Parts(ctx context.Context, database, table string) ([]PartInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			name, active, partition,
			rows, bytes_on_disk, data_compressed_bytes, data_uncompressed_bytes,
			marks_bytes, modification_time, remove_time,
			level, data_version, min_date, max_date,
			engine, disk_name, path
		FROM system.parts
		WHERE database = $1 AND table = $2 AND active
		ORDER BY modification_time DESC, name
	`, database, table)
	if err != nil {
		return nil, fmt.Errorf("query system.parts: %w", err)
	}
	defer rows.Close()

	var out []PartInfo
	for rows.Next() {
		var p PartInfo
		var active uint8
		if err := rows.Scan(
			&p.Name, &active, &p.Partition,
			&p.Rows, &p.BytesOnDisk, &p.DataCompressed, &p.DataUncompressed,
			&p.MarksBytes, &p.ModificationTime, &p.RemovalTime,
			&p.Level, &p.DataVersion, &p.MinDate, &p.MaxDate,
			&p.Engine, &p.DiskName, &p.Path,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		p.Active = active == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

// CompressionRatio returns the inverse compression factor: uncompressed /
// compressed. For example, 5.0 means the part shrinks 5x on disk.
// Returns 0 when uncompressed bytes are 0.
func (p PartInfo) CompressionRatio() float64 {
	if p.DataCompressed == 0 {
		return 0
	}
	return float64(p.DataUncompressed) / float64(p.DataCompressed)
}
