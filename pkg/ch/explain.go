package ch

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ExplainMode is one of the modes ClickHouse's EXPLAIN supports. Only the
// four most useful to chtop are exposed.
type ExplainMode string

const (
	ExplainPlan     ExplainMode = "PLAN"
	ExplainPipeline ExplainMode = "PIPELINE"
	ExplainSyntax   ExplainMode = "SYNTAX"
	ExplainEstimate ExplainMode = "ESTIMATE"
)

// ErrQueryNotFound is returned by ExplainQueryID when the query_id has no
// terminating event in system.query_log (already aged out, or never
// recorded).
var ErrQueryNotFound = errors.New("query_id not found in system.query_log")

// ExplainQueryID fetches the rendered query text for queryID and runs
// EXPLAIN <mode> against it, returning the result as a single string with
// one EXPLAIN row per line.
func (c *Client) ExplainQueryID(
	ctx context.Context, queryID string, mode ExplainMode,
) (string, error) {
	text, err := c.queryTextByID(ctx, queryID)
	if err != nil {
		return "", err
	}
	return c.ExplainText(ctx, text, mode)
}

// ExplainText runs EXPLAIN <mode> <text> directly.
func (c *Client) ExplainText(
	ctx context.Context, text string, mode ExplainMode,
) (string, error) {
	if !isKnownExplainMode(mode) {
		return "", fmt.Errorf("unsupported explain mode %q", mode)
	}
	sql := explainSQL(mode, text)
	rows, err := c.conn.Query(c.tagged(ctx), sql)
	if err != nil {
		return "", fmt.Errorf("explain: %w", err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return "", fmt.Errorf("scan explain: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("explain rows: %w", err)
	}
	return strings.Join(lines, "\n"), nil
}

func (c *Client) queryTextByID(ctx context.Context, queryID string) (string, error) {
	row := c.conn.QueryRow(c.tagged(ctx), `
		SELECT query
		FROM system.query_log
		WHERE query_id = $1 AND type IN (2,3,4)
		ORDER BY event_time DESC
		LIMIT 1
	`, queryID)
	var text string
	if err := row.Scan(&text); err != nil {
		if errors.Is(err, errNoRowsSentinel()) {
			return "", ErrQueryNotFound
		}
		return "", fmt.Errorf("lookup query: %w", err)
	}
	return text, nil
}

// errNoRowsSentinel returns the driver's sql.ErrNoRows without pulling
// database/sql into this package's exported surface. clickhouse-go also uses
// it, exposed through QueryRow.Scan when there are no rows.
func errNoRowsSentinel() error {
	// kept indirect on purpose so this file's import set stays minimal.
	return errors.New("sql: no rows in result set")
}

func isKnownExplainMode(m ExplainMode) bool {
	switch m {
	case ExplainPlan, ExplainPipeline, ExplainSyntax, ExplainEstimate:
		return true
	}
	return false
}

// explainSQL composes the EXPLAIN statement. Pure: no driver, no context.
// The query text is embedded verbatim — system.query_log already stores the
// rendered text with parameter substitution done.
func explainSQL(mode ExplainMode, text string) string {
	return fmt.Sprintf("EXPLAIN %s %s", mode, strings.TrimSpace(text))
}
