package ch

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// MaterializedViewInfo collapses the bits needed to render an MV in the
// graph view: where it reads from, where it writes to, and the target's
// current size if the target is a real table.
type MaterializedViewInfo struct {
	Database    string
	Name        string
	Source      string // best-effort: parsed from as_select
	Target      string // best-effort: parsed from create_table_query (TO clause)
	TargetRows  uint64
	TargetBytes uint64
}

// MaterializedViews lists every MaterializedView visible to the user with
// source / target parsed from DDL and target row/byte counts when the target
// is a regular table.
func (c *Client) MaterializedViews(ctx context.Context) ([]MaterializedViewInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			t.database, t.name, t.as_select, t.create_table_query
		FROM system.tables AS t
		WHERE t.engine = 'MaterializedView'
		ORDER BY t.database, t.name
	`)
	if err != nil {
		return nil, fmt.Errorf("query system.tables: %w", err)
	}
	defer rows.Close()

	var out []MaterializedViewInfo
	for rows.Next() {
		var info MaterializedViewInfo
		var asSelect, createQuery string
		if err := rows.Scan(&info.Database, &info.Name, &asSelect, &createQuery); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		info.Source, info.Target = parseMVSourceTarget(asSelect, createQuery)
		out = append(out, info)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Best-effort sizes: look up each target in system.parts.
	for i := range out {
		if out[i].Target == "" {
			continue
		}
		db, table := splitQualified(out[i].Target)
		if table == "" {
			continue
		}
		rs, err := c.conn.Query(c.tagged(ctx), `
			SELECT coalesce(sum(rows), 0), coalesce(sum(bytes_on_disk), 0)
			FROM system.parts
			WHERE database = $1 AND table = $2 AND active
		`, db, table)
		if err != nil {
			continue
		}
		if rs.Next() {
			_ = rs.Scan(&out[i].TargetRows, &out[i].TargetBytes)
		}
		_ = rs.Close()
	}

	return out, nil
}

// splitQualified splits "db.table" into ("db", "table"). When the input has
// no dot the whole thing is treated as the table in the default database
// (returned as empty db).
func splitQualified(name string) (string, string) {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i], name[i+1:]
	}
	return "default", name
}

var (
	// fromIdentRe matches the table identifier after FROM. Allows backtick
	// quoting, dots, dashes and word chars. Captures the identifier text.
	fromIdentRe = regexp.MustCompile(
		"(?is)\\bFROM\\s+(`[^`]+`(?:\\.`[^`]+`)?|[A-Za-z_][A-Za-z0-9_]*(?:\\.[A-Za-z_][A-Za-z0-9_]*)?)",
	)
	// toIdentRe matches the table identifier after TO in a CREATE MV.
	toIdentRe = regexp.MustCompile(
		"(?is)\\bTO\\s+(`[^`]+`(?:\\.`[^`]+`)?|[A-Za-z_][A-Za-z0-9_]*(?:\\.[A-Za-z_][A-Za-z0-9_]*)?)",
	)
)

// parseMVSourceTarget extracts a (source, target) pair from the materialized
// view's as_select (for source) and create_table_query (for target). Pure,
// regex-based, intentionally simple — covers the common shapes and degrades
// to empty strings on anything weird so the view stays usable.
func parseMVSourceTarget(asSelect, createQuery string) (source, target string) {
	if m := fromIdentRe.FindStringSubmatch(asSelect); len(m) == 2 {
		source = unquoteIdent(m[1])
	}
	if m := toIdentRe.FindStringSubmatch(createQuery); len(m) == 2 {
		target = unquoteIdent(m[1])
	}
	return source, target
}

// unquoteIdent strips backticks from a single identifier or db.table pair.
func unquoteIdent(s string) string {
	return strings.ReplaceAll(s, "`", "")
}
