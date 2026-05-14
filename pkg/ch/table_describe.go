package ch

import (
	"context"
	"fmt"
)

// TableDescription captures every system.tables column the chtop detail
// header cares about. Strings can be empty when ClickHouse hasn't recorded
// the field (e.g. non-MergeTree engines don't have a sorting key).
type TableDescription struct {
	Database         string
	Name             string
	Engine           string
	EngineFull       string
	CreateTableQuery string
	PartitionKey     string
	SortingKey       string
	PrimaryKey       string
	SamplingKey      string
	StoragePolicy    string
	TTLExpression    string
	Comment          string
}

// DescribeTable returns the row of system.tables for one (database, table).
// Returns an error wrapping sql.ErrNoRows if the table doesn't exist or the
// caller can't see it.
func (c *Client) DescribeTable(
	ctx context.Context, database, table string,
) (TableDescription, error) {
	row := c.conn.QueryRow(c.tagged(ctx), `
		SELECT
			database, name, engine, engine_full,
			create_table_query,
			partition_key, sorting_key, primary_key, sampling_key,
			storage_policy, total_marks, comment
		FROM system.tables
		WHERE database = $1 AND name = $2
	`, database, table)

	var d TableDescription
	var totalMarks uint64 // scanned but currently unused; kept so the column list mirrors what we may surface later
	if err := row.Scan(
		&d.Database, &d.Name, &d.Engine, &d.EngineFull,
		&d.CreateTableQuery,
		&d.PartitionKey, &d.SortingKey, &d.PrimaryKey, &d.SamplingKey,
		&d.StoragePolicy, &totalMarks, &d.Comment,
	); err != nil {
		return TableDescription{}, fmt.Errorf("scan system.tables: %w", err)
	}
	return d, nil
}
