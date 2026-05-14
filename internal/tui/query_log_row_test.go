package tui

import (
	"testing"
	"time"

	"github.com/sonirico/chtop/pkg/ch"
	"github.com/stretchr/testify/require"
)

func TestQueryLogRow(t *testing.T) {
	t.Parallel()
	ts := time.Date(2026, 5, 14, 10, 30, 45, 0, time.UTC)
	type testCase struct {
		name string
		q    ch.QueryLogInfo
		want []string
	}
	cases := []testCase{
		{
			name: "successful select",
			q: ch.QueryLogInfo{
				EventTime:   ts,
				User:        "alice",
				Database:    "analytics",
				DurationMs:  4200,
				MemoryUsage: 2 * 1024 * 1024,
				ReadRows:    250_000,
				Type:        ch.QueryTypeFinish,
				Query:       "SELECT count() FROM events",
			},
			want: []string{
				"10:30:45", "alice", "analytics",
				"4.2s", "2.00 MiB", "250.00K", "ok",
				"SELECT count() FROM events",
			},
		},
		{
			name: "exception before start",
			q: ch.QueryLogInfo{
				EventTime: ts,
				User:      "bob",
				Database:  "default",
				Type:      ch.QueryTypeExceptionBeforeStart,
				Query:     "SELECT * FROM nope",
			},
			want: []string{
				"10:30:45", "bob", "default",
				"0ms", "0 B", "0", "err",
				"SELECT * FROM nope",
			},
		},
		{
			name: "exception while processing",
			q: ch.QueryLogInfo{
				EventTime: ts,
				User:      "carol",
				Database:  "default",
				Type:      ch.QueryTypeExceptionWhileProcessing,
				Query:     "SELECT 1/0",
			},
			want: []string{
				"10:30:45", "carol", "default",
				"0ms", "0 B", "0", "err",
				"SELECT 1/0",
			},
		},
		{
			name: "multiline query collapses",
			q: ch.QueryLogInfo{
				EventTime:  ts,
				User:       "dave",
				Database:   "default",
				DurationMs: 1500,
				Type:       ch.QueryTypeFinish,
				Query:      "SELECT *\n  FROM t\n  WHERE x=1",
			},
			want: []string{
				"10:30:45", "dave", "default",
				"1.5s", "0 B", "0", "ok",
				"SELECT * FROM t WHERE x=1",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, queryLogRow(tc.q))
		})
	}
}
