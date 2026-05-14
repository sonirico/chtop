package ch

import (
	"context"
	"fmt"
	"time"
)

// ErrorInfo is one row of system.errors: a server-wide error counter keyed
// by error name. Useful for spotting trouble at a glance.
type ErrorInfo struct {
	Name             string
	Code             int32
	Value            uint64
	LastErrorTime    time.Time
	LastErrorMessage string
	Remote           bool
}

// Errors lists every entry of system.errors, ordered by Value desc so the
// noisiest errors are first.
func (c *Client) Errors(ctx context.Context) ([]ErrorInfo, error) {
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			name, code, value,
			last_error_time, last_error_message,
			remote
		FROM system.errors
		ORDER BY value DESC, name
	`)
	if err != nil {
		return nil, fmt.Errorf("query system.errors: %w", err)
	}
	defer rows.Close()

	var out []ErrorInfo
	for rows.Next() {
		var e ErrorInfo
		var remote uint8
		if err := rows.Scan(
			&e.Name, &e.Code, &e.Value,
			&e.LastErrorTime, &e.LastErrorMessage,
			&remote,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		e.Remote = remote == 1
		out = append(out, e)
	}
	return out, rows.Err()
}
