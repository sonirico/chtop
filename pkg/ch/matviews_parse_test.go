package ch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMVSourceTarget(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name        string
		asSelect    string
		createQuery string
		wantSource  string
		wantTarget  string
	}
	cases := []testCase{
		{
			name:        "to clause with db qualifier",
			asSelect:    "SELECT user_id, count() FROM events.raw GROUP BY user_id",
			createQuery: "CREATE MATERIALIZED VIEW analytics.user_counts_mv TO analytics.user_counts AS SELECT user_id, count() FROM events.raw GROUP BY user_id",
			wantSource:  "events.raw",
			wantTarget:  "analytics.user_counts",
		},
		{
			name:        "to clause without db qualifier",
			asSelect:    "SELECT * FROM users",
			createQuery: "CREATE MATERIALIZED VIEW v TO users_target AS SELECT * FROM users",
			wantSource:  "users",
			wantTarget:  "users_target",
		},
		{
			name:        "no TO clause yields empty target",
			asSelect:    "SELECT * FROM analytics.events",
			createQuery: "CREATE MATERIALIZED VIEW analytics.events_mv ENGINE = AggregatingMergeTree ORDER BY user_id AS SELECT * FROM analytics.events",
			wantSource:  "analytics.events",
			wantTarget:  "",
		},
		{
			name:        "backticked identifiers",
			asSelect:    "SELECT * FROM `db.with.dots`.`table-name`",
			createQuery: "CREATE MATERIALIZED VIEW v TO `analytics`.`out` AS SELECT * FROM `db.with.dots`.`table-name`",
			wantSource:  "db.with.dots.table-name",
			wantTarget:  "analytics.out",
		},
		{
			name:        "from with alias",
			asSelect:    "SELECT a.x FROM events.raw AS a",
			createQuery: "CREATE MATERIALIZED VIEW v TO out AS SELECT a.x FROM events.raw AS a",
			wantSource:  "events.raw",
			wantTarget:  "out",
		},
		{
			name:        "no FROM at all",
			asSelect:    "SELECT 1",
			createQuery: "CREATE MATERIALIZED VIEW v AS SELECT 1",
			wantSource:  "",
			wantTarget:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotSource, gotTarget := parseMVSourceTarget(tc.asSelect, tc.createQuery)
			require.Equal(t, tc.wantSource, gotSource, "source")
			require.Equal(t, tc.wantTarget, gotTarget, "target")
		})
	}
}
