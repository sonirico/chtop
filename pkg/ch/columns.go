package ch

import (
	"context"
	"fmt"
)

// ColumnInfo describes one column of a table: name, type, optional custom
// codec, and disk usage stats from system.parts_columns aggregated across
// active parts. The schema tab of the table detail view consumes this.
type ColumnInfo struct {
	Name              string
	Type              string
	DefaultKind       string
	DefaultExpression string
	Comment           string
	CompressionCodec  string
	IsInPartitionKey  bool
	IsInSortingKey    bool
	IsInPrimaryKey    bool
	DataCompressed    uint64
	DataUncompressed  uint64
}

// Columns returns the column definitions for a table, plus aggregated
// per-column disk usage across active parts. Order matches the table's
// definition (position).
func (c *Client) Columns(ctx context.Context, database, table string) ([]ColumnInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			c.name,
			c.type,
			c.default_kind,
			c.default_expression,
			c.comment,
			c.compression_codec,
			c.is_in_partition_key,
			c.is_in_sorting_key,
			c.is_in_primary_key,
			coalesce(sum(pc.data_compressed_bytes),   0) AS compressed_bytes,
			coalesce(sum(pc.data_uncompressed_bytes), 0) AS uncompressed_bytes
		FROM system.columns AS c
		LEFT JOIN system.parts_columns AS pc
			ON pc.database = c.database
			AND pc.table   = c.table
			AND pc.column  = c.name
			AND pc.active
		WHERE c.database = $1 AND c.table = $2
		GROUP BY
			c.name, c.type, c.default_kind, c.default_expression,
			c.comment, c.compression_codec,
			c.is_in_partition_key, c.is_in_sorting_key, c.is_in_primary_key,
			c.position
		ORDER BY c.position
	`, database, table)
	if err != nil {
		return nil, fmt.Errorf("query system.columns: %w", err)
	}
	defer rows.Close()

	var out []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var inPart, inSort, inPrim uint8
		if err := rows.Scan(
			&col.Name, &col.Type, &col.DefaultKind, &col.DefaultExpression,
			&col.Comment, &col.CompressionCodec,
			&inPart, &inSort, &inPrim,
			&col.DataCompressed, &col.DataUncompressed,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		col.IsInPartitionKey = inPart == 1
		col.IsInSortingKey = inSort == 1
		col.IsInPrimaryKey = inPrim == 1
		out = append(out, col)
	}
	return out, rows.Err()
}

// CompressionRatio returns uncompressed / compressed for this column. 0 when
// no data is on disk yet. Useful to spot which columns are the most / least
// compressible.
func (c ColumnInfo) CompressionRatio() float64 {
	if c.DataCompressed == 0 {
		return 0
	}
	return float64(c.DataUncompressed) / float64(c.DataCompressed)
}
