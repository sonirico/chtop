package tui

import (
	"testing"
	"time"

	"github.com/sonirico/chtop/pkg/ch"
	"github.com/stretchr/testify/require"
)

func TestMutationRow(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 5, 14, 9, 0, 0, 0, time.UTC)
	type testCase struct {
		name string
		m    ch.MutationInfo
		want []string
	}
	cases := []testCase{
		{
			name: "in-progress mutation",
			m: ch.MutationInfo{
				MutationID: "0000000123",
				Command:    "DELETE WHERE user_id = 42",
				CreateTime: created,
				PartsToDo:  17,
				IsDone:     false,
			},
			want: []string{
				"0000000123", "DELETE WHERE user_id = 42", "running",
				"2026-05-14 09:00:00", "17", "",
			},
		},
		{
			name: "finished mutation",
			m: ch.MutationInfo{
				MutationID: "0000000099",
				Command:    "MATERIALIZE TTL",
				CreateTime: created,
				PartsToDo:  0,
				IsDone:     true,
			},
			want: []string{
				"0000000099", "MATERIALIZE TTL", "done",
				"2026-05-14 09:00:00", "0", "",
			},
		},
		{
			name: "failed mutation surfaces fail reason",
			m: ch.MutationInfo{
				MutationID:       "0000000200",
				Command:          "UPDATE col = 1 WHERE bad",
				CreateTime:       created,
				PartsToDo:        5,
				IsDone:           false,
				LatestFailReason: "Cannot parse expression",
			},
			want: []string{
				"0000000200", "UPDATE col = 1 WHERE bad", "running",
				"2026-05-14 09:00:00", "5", "Cannot parse expression",
			},
		},
		{
			name: "multiline command collapses to single line",
			m: ch.MutationInfo{
				MutationID: "0000000300",
				Command:    "DELETE\n  WHERE\n  ts < now()",
				CreateTime: created,
				PartsToDo:  3,
			},
			want: []string{
				"0000000300", "DELETE   WHERE   ts < now()", "running",
				"2026-05-14 09:00:00", "3", "",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, mutationRow(tc.m))
		})
	}
}
