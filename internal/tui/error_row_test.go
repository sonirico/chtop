package tui

import (
	"testing"
	"time"

	"github.com/sonirico/chtop/pkg/ch"
	"github.com/stretchr/testify/require"
)

func TestErrorRow(t *testing.T) {
	t.Parallel()
	seen := time.Date(2026, 5, 14, 12, 0, 0, 0, time.UTC)
	type testCase struct {
		name string
		e    ch.ErrorInfo
		want []string
	}
	cases := []testCase{
		{
			name: "common local error",
			e: ch.ErrorInfo{
				Name:             "CANNOT_PARSE_DATETIME",
				Code:             41,
				Value:            12_345,
				LastErrorTime:    seen,
				LastErrorMessage: "Cannot parse '...' as DateTime",
				Remote:           false,
			},
			want: []string{
				"CANNOT_PARSE_DATETIME", "41", "12.35K",
				"2026-05-14 12:00:00", "local",
				"Cannot parse '...' as DateTime",
			},
		},
		{
			name: "remote error",
			e: ch.ErrorInfo{
				Name:             "NETWORK_ERROR",
				Code:             209,
				Value:            7,
				LastErrorTime:    seen,
				LastErrorMessage: "Connection refused",
				Remote:           true,
			},
			want: []string{
				"NETWORK_ERROR", "209", "7",
				"2026-05-14 12:00:00", "remote",
				"Connection refused",
			},
		},
		{
			name: "multiline message collapses",
			e: ch.ErrorInfo{
				Name:             "EXAMPLE",
				Code:             1,
				Value:            1,
				LastErrorTime:    seen,
				LastErrorMessage: "first line\nsecond line\nthird",
			},
			want: []string{
				"EXAMPLE", "1", "1",
				"2026-05-14 12:00:00", "local",
				"first line second line third",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, errorRow(tc.e))
		})
	}
}

func TestIsGrowing(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		prev uint64
		curr uint64
		want bool
	}
	cases := []testCase{
		{"unchanged", 100, 100, false},
		{"increased by one", 100, 101, true},
		{"newly seen", 0, 5, true},
		{"counter reset (server restart)", 500, 10, false},
		{"both zero", 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isGrowing(tc.prev, tc.curr))
		})
	}
}
