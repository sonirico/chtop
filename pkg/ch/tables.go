package ch

import (
	"context"
	"fmt"
)

// TableInfo summarises one row of system.tables enriched with size and row
// counts from system.parts. CompressedBytes / UncompressedBytes come from
// the column data on disk and let you compute the compression ratio.
type TableInfo struct {
	Database          string
	Name              string
	Engine            string
	IsTemporary       bool
	TotalRows         uint64
	TotalBytes        uint64 // bytes_on_disk
	CompressedBytes   uint64 // data_compressed_bytes
	UncompressedBytes uint64 // data_uncompressed_bytes
	PartCount         uint64
}

// CompressionRatio returns compressed/uncompressed. 0 means unknown (no
// uncompressed bytes reported). Smaller is better; 0.20 means 5x compression.
func (t TableInfo) CompressionRatio() float64 {
	if t.UncompressedBytes == 0 {
		return 0
	}
	return float64(t.CompressedBytes) / float64(t.UncompressedBytes)
}

// ListTables returns every table the user can see, with size + part count
// aggregated from system.parts.active. System databases are filtered out by
// default; pass includeSystem=true to keep them.
func (c *Client) ListTables(ctx context.Context, includeSystem bool) ([]TableInfo, error) {
	query := `
		SELECT
			t.database,
			t.name,
			t.engine,
			t.is_temporary,
			coalesce(p.total_rows, 0)        AS total_rows,
			coalesce(p.total_bytes, 0)       AS total_bytes,
			coalesce(p.compressed_bytes, 0)  AS compressed_bytes,
			coalesce(p.uncompressed_bytes,0) AS uncompressed_bytes,
			coalesce(p.part_count, 0)        AS part_count
		FROM system.tables AS t
		LEFT JOIN (
			SELECT database, table,
			       sum(rows)                    AS total_rows,
			       sum(bytes_on_disk)           AS total_bytes,
			       sum(data_compressed_bytes)   AS compressed_bytes,
			       sum(data_uncompressed_bytes) AS uncompressed_bytes,
			       count()                      AS part_count
			FROM system.parts
			WHERE active
			GROUP BY database, table
		) AS p ON p.database = t.database AND p.table = t.name
	`
	if !includeSystem {
		query += " WHERE t.database NOT IN ('system','INFORMATION_SCHEMA','information_schema')"
	}
	query += " ORDER BY t.database, t.name"

	rows, err := c.conn.Query(c.tagged(ctx), query)
	if err != nil {
		return nil, fmt.Errorf("query system.tables: %w", err)
	}
	defer rows.Close()

	var out []TableInfo
	for rows.Next() {
		var t TableInfo
		var isTemp uint8
		if err := rows.Scan(
			&t.Database, &t.Name, &t.Engine, &isTemp,
			&t.TotalRows, &t.TotalBytes,
			&t.CompressedBytes, &t.UncompressedBytes, &t.PartCount,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		t.IsTemporary = isTemp == 1
		out = append(out, t)
	}
	return out, rows.Err()
}
