package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHumanBytes(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		n    int64
		want string
	}
	cases := []testCase{
		{"zero", 0, "0 B"},
		{"negative collapses to zero", -1, "0 B"},
		{"under one KiB", 512, "512 B"},
		{"one KiB exact", 1024, "1.00 KiB"},
		{"one and a half KiB", 1536, "1.50 KiB"},
		{"one MiB exact", 1024 * 1024, "1.00 MiB"},
		{"one GiB exact", 1024 * 1024 * 1024, "1.00 GiB"},
		{"one TiB exact", 1024 * 1024 * 1024 * 1024, "1.00 TiB"},
		{"one PiB exact", 1024 * 1024 * 1024 * 1024 * 1024, "1.00 PiB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, humanBytes(tc.n))
		})
	}
}

func TestHumanCount(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		n    uint64
		want string
	}
	cases := []testCase{
		{"zero", 0, "0"},
		{"single digit", 7, "7"},
		{"sub-thousand", 999, "999"},
		{"one thousand", 1_000, "1.00K"},
		{"ten thousand", 12_500, "12.50K"},
		{"one million", 1_000_000, "1.00M"},
		{"hundred million", 250_000_000, "250.00M"},
		{"one billion", 1_000_000_000, "1.00B"},
		{"large billion", 4_200_000_000, "4.20B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, humanCount(tc.n))
		})
	}
}

func TestHumanDuration(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		d    time.Duration
		want string
	}
	cases := []testCase{
		{"zero", 0, "0ms"},
		{"sub-second milliseconds", 250 * time.Millisecond, "250ms"},
		{"one second", time.Second, "1.0s"},
		{"seconds with fraction", 2500 * time.Millisecond, "2.5s"},
		{"minutes and seconds", 90 * time.Second, "1m30s"},
		{"hours and minutes", 2*time.Hour + 15*time.Minute, "2h15m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, humanDuration(tc.d))
		})
	}
}
