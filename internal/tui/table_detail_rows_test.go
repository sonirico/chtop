package tui

import (
	"testing"
	"time"

	"github.com/sonirico/chtop/pkg/ch"
	"github.com/stretchr/testify/require"
)

func TestColumnRow(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		col  ch.ColumnInfo
		want []string
	}
	cases := []testCase{
		{
			name: "plain column without codec or keys",
			col: ch.ColumnInfo{
				Name:             "event_id",
				Type:             "UUID",
				CompressionCodec: "",
				DataCompressed:   1024,
				DataUncompressed: 2048,
			},
			want: []string{"event_id", "UUID", "default", "1.00 KiB", "2.0x", ""},
		},
		{
			name: "column with custom codec, in primary key",
			col: ch.ColumnInfo{
				Name:             "ts",
				Type:             "DateTime",
				CompressionCodec: "CODEC(DoubleDelta, LZ4)",
				DataCompressed:   200,
				DataUncompressed: 1000,
				IsInPartitionKey: true,
				IsInPrimaryKey:   true,
				IsInSortingKey:   true,
			},
			want: []string{
				"ts",
				"DateTime",
				"CODEC(DoubleDelta, LZ4)",
				"200 B",
				"5.0x",
				"PK PART SORT",
			},
		},
		{
			name: "column with no on-disk data",
			col: ch.ColumnInfo{
				Name: "rare",
				Type: "Nullable(String)",
			},
			want: []string{"rare", "Nullable(String)", "default", "0 B", "-", ""},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, columnRow(tc.col))
		})
	}
}

func TestPartRow(t *testing.T) {
	t.Parallel()
	mod := time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC)
	type testCase struct {
		name string
		p    ch.PartInfo
		want []string
	}
	cases := []testCase{
		{
			name: "typical merge-tree part",
			p: ch.PartInfo{
				Name:             "202605_1_42_3",
				Partition:        "202605",
				Rows:             1500000,
				BytesOnDisk:      4 * 1024 * 1024,
				DataCompressed:   2 * 1024 * 1024,
				DataUncompressed: 10 * 1024 * 1024,
				Level:            3,
				ModificationTime: mod,
			},
			want: []string{
				"202605_1_42_3", "202605", "1.50M",
				"4.00 MiB", "5.0x", "3", "2026-05-14 10:30:00",
			},
		},
		{
			name: "empty part still renders zeros",
			p: ch.PartInfo{
				Name:             "x_0_0_0",
				Partition:        "x",
				ModificationTime: mod,
			},
			want: []string{"x_0_0_0", "x", "0", "0 B", "-", "0", "2026-05-14 10:30:00"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, partRow(tc.p))
		})
	}
}
