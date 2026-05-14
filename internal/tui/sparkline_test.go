package tui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSparkline(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name   string
		values []float64
		width  int
		want   string
	}
	cases := []testCase{
		{"empty values returns empty", nil, 10, ""},
		{"zero width returns empty", []float64{1, 2, 3}, 0, ""},
		{"all zeros render as flat baseline", []float64{0, 0, 0}, 5, "▁▁▁"},
		{"single max value", []float64{5}, 5, "█"},
		{
			name:   "monotonic ramp picks every block",
			values: []float64{0, 1, 2, 3, 4, 5, 6, 7},
			width:  8,
			// 0/7 ratio -> ▁, 1/7 -> ▂, ..., 7/7 -> █
			want: "▁▂▃▄▅▆▇█",
		},
		{
			name:   "values longer than width keep the tail",
			values: []float64{99, 99, 99, 1, 2, 3},
			width:  3,
			want:   "▃▅█",
		},
		{
			name:   "constant series renders as a flat top",
			values: []float64{7, 7, 7, 7},
			width:  4,
			want:   "████",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, sparkline(tc.values, tc.width))
		})
	}
}
