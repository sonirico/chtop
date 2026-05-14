package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPerSecondRate(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name    string
		prev    int64
		curr    int64
		elapsed time.Duration
		want    float64
	}
	cases := []testCase{
		{"no elapsed time", 100, 200, 0, 0},
		{"negative elapsed treated as no rate", 100, 200, -time.Second, 0},
		{"counter reset (curr < prev) returns zero", 500, 100, time.Second, 0},
		{"one second delta", 100, 200, time.Second, 100},
		{"half a second", 100, 300, 500 * time.Millisecond, 400},
		{"five seconds, slow rate", 100, 150, 5 * time.Second, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.InDelta(t, tc.want, perSecondRate(tc.prev, tc.curr, tc.elapsed), 1e-9)
		})
	}
}

func TestFormatMetric(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name  string
		key   string
		value int64
		want  string
	}
	cases := []testCase{
		{"memory tracked formats as bytes", "MemoryTracking", 1024 * 1024 * 1024, "1.00 GiB"},
		{"memory resident formats as bytes", "MemoryResident", 512 * 1024 * 1024, "512.00 MiB"},
		{"uptime formats as duration", "Uptime", 90, "1m30s"},
		{"unknown metric falls back to humanCount", "Query", 1500, "1.50K"},
		{"unknown metric small int stays raw", "Merge", 4, "4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, formatMetric(tc.key, tc.value))
		})
	}
}
