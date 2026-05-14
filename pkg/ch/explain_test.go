package ch

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExplainSQL(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		mode ExplainMode
		text string
		want string
	}
	cases := []testCase{
		{
			name: "plan with simple select",
			mode: ExplainPlan,
			text: "SELECT 1",
			want: "EXPLAIN PLAN SELECT 1",
		},
		{
			name: "pipeline with multiline preserves inner newlines",
			mode: ExplainPipeline,
			text: "SELECT *\nFROM t\nWHERE x = 1",
			want: "EXPLAIN PIPELINE SELECT *\nFROM t\nWHERE x = 1",
		},
		{
			name: "trims outer whitespace",
			mode: ExplainSyntax,
			text: "   SELECT 1   \n",
			want: "EXPLAIN SYNTAX SELECT 1",
		},
		{
			name: "estimate",
			mode: ExplainEstimate,
			text: "SELECT count() FROM events",
			want: "EXPLAIN ESTIMATE SELECT count() FROM events",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, explainSQL(tc.mode, tc.text))
		})
	}
}

func TestIsKnownExplainMode(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name string
		mode ExplainMode
		want bool
	}
	cases := []testCase{
		{"plan ok", ExplainPlan, true},
		{"pipeline ok", ExplainPipeline, true},
		{"syntax ok", ExplainSyntax, true},
		{"estimate ok", ExplainEstimate, true},
		{"empty rejected", "", false},
		{"unknown rejected", "AST", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isKnownExplainMode(tc.mode))
		})
	}
}
