package ch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartInfo_CompressionRatio(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		part PartInfo
		want float64
	}
	cases := []testCase{
		{
			name: "no data yet",
			part: PartInfo{DataCompressed: 0, DataUncompressed: 0},
			want: 0,
		},
		{
			name: "zero compressed returns zero to avoid div-by-zero",
			part: PartInfo{DataCompressed: 0, DataUncompressed: 100},
			want: 0,
		},
		{
			name: "five times",
			part: PartInfo{DataCompressed: 100, DataUncompressed: 500},
			want: 5,
		},
		{
			name: "incompressible content reports 1x",
			part: PartInfo{DataCompressed: 200, DataUncompressed: 200},
			want: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.InDelta(t, tc.want, tc.part.CompressionRatio(), 1e-9)
		})
	}
}

func TestColumnInfo_CompressionRatio(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		col  ColumnInfo
		want float64
	}
	cases := []testCase{
		{
			name: "no data yet",
			col:  ColumnInfo{},
			want: 0,
		},
		{
			name: "ten times",
			col:  ColumnInfo{DataCompressed: 10, DataUncompressed: 100},
			want: 10,
		},
		{
			name: "half a percent over 1x",
			col:  ColumnInfo{DataCompressed: 200, DataUncompressed: 201},
			want: 1.005,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.InDelta(t, tc.want, tc.col.CompressionRatio(), 1e-9)
		})
	}
}
