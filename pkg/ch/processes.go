package ch

import (
	"context"
	"fmt"
	"time"
)

// Process is one row of system.processes: a currently-running query on this
// replica. QueryID is what KillQuery takes.
type Process struct {
	QueryID         string
	User            string
	ClientHostname  string
	InitialUser     string
	Address         string
	Elapsed         time.Duration
	ReadRows        uint64
	ReadBytes       uint64
	TotalRows       uint64
	MemoryUsage     int64
	PeakMemoryUsage int64
	Query           string
}

// Processes returns the currently running queries. Sort is by elapsed time
// desc so the worst offenders are on top.
func (c *Client) Processes(ctx context.Context) ([]Process, error) {
	// Filter out our own polling queries so the view doesn't churn on itself.
	rows, err := c.conn.Query(c.tagged(ctx), `
		SELECT
			query_id, user, client_hostname, initial_user, address,
			elapsed, read_rows, read_bytes, total_rows_approx,
			memory_usage, peak_memory_usage, query
		FROM system.processes
		WHERE query_id NOT LIKE concat($1, '%')
		ORDER BY elapsed DESC
	`, QueryIDPrefix)
	if err != nil {
		return nil, fmt.Errorf("query system.processes: %w", err)
	}
	defer rows.Close()

	var out []Process
	for rows.Next() {
		var p Process
		var elapsedSec float64
		var address string
		if err := rows.Scan(
			&p.QueryID, &p.User, &p.ClientHostname, &p.InitialUser, &address,
			&elapsedSec, &p.ReadRows, &p.ReadBytes, &p.TotalRows,
			&p.MemoryUsage, &p.PeakMemoryUsage, &p.Query,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		p.Address = address
		p.Elapsed = time.Duration(elapsedSec * float64(time.Second))
		out = append(out, p)
	}
	return out, rows.Err()
}

// KillQuery asks the server to cancel a running query. Async on the server
// side: the query may take a few seconds to actually stop after the call
// returns. Returns nil if the query id is no longer running.
func (c *Client) KillQuery(ctx context.Context, queryID string) error {
	q := fmt.Sprintf("KILL QUERY WHERE query_id = '%s' SYNC", escapeSingleQuotes(queryID))
	if err := c.conn.Exec(c.tagged(ctx), q); err != nil {
		return fmt.Errorf("kill query: %w", err)
	}
	return nil
}

func escapeSingleQuotes(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\\', '\'')
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}
