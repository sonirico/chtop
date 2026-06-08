package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestSparklineColored(t *testing.T) {
	t.Parallel()

	t.Run("empty inputs render nothing", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, "", sparklineColored(nil, 10))
		require.Equal(t, "", sparklineColored([]float64{1, 2}, 0))
	})

	t.Run("packs two samples per braille cell", func(t *testing.T) {
		t.Parallel()
		// 6 samples -> 3 cells. Strip ANSI to count the braille runes.
		out := stripANSI(sparklineColored([]float64{0, 1, 2, 3, 4, 5}, 10))
		require.Equal(t, 3, utf8.RuneCountInString(out))
		for _, r := range out {
			require.GreaterOrEqual(t, r, rune(brailleBase))
			require.Less(t, r, rune(brailleBase+0x100))
		}
	})

	t.Run("keeps only the trailing 2*width samples", func(t *testing.T) {
		t.Parallel()
		vals := make([]float64, 100)
		out := stripANSI(sparklineColored(vals, 4))
		require.Equal(t, 4, utf8.RuneCountInString(out))
	})
}

func TestAreaGraph(t *testing.T) {
	t.Parallel()

	t.Run("returns h empty rows when no values", func(t *testing.T) {
		t.Parallel()
		rows := areaGraph(nil, 10, 3)
		require.Len(t, rows, 3)
		for _, r := range rows {
			require.Equal(t, strings.Repeat(" ", 10), r)
		}
	})

	t.Run("returns h rows each w runes wide", func(t *testing.T) {
		t.Parallel()
		rows := areaGraph([]float64{1, 2, 3, 4, 5}, 5, 4)
		require.Len(t, rows, 4)
		for _, r := range rows {
			require.Equal(t, 5, utf8.RuneCountInString(r))
		}
	})

	t.Run("peak column is fully filled at bottom row", func(t *testing.T) {
		t.Parallel()
		// With a single max-value sample the bottom row must be '█' at that column.
		rows := areaGraph([]float64{1.0}, 1, 2)
		require.Len(t, rows, 2)
		// Bottom row (index 1) must contain a full block.
		require.Equal(t, "█", rows[1])
	})

	t.Run("zero values produce all spaces", func(t *testing.T) {
		t.Parallel()
		rows := areaGraph([]float64{0, 0, 0}, 3, 2)
		for _, r := range rows {
			require.Equal(t, strings.Repeat(" ", 3), r)
		}
	})

	t.Run("width 0 returns empty rows", func(t *testing.T) {
		t.Parallel()
		rows := areaGraph([]float64{1, 2, 3}, 0, 3)
		require.Len(t, rows, 3)
		for _, r := range rows {
			require.Equal(t, "", r)
		}
	})
}

func TestSeriesStats(t *testing.T) {
	t.Parallel()

	t.Run("empty series is all zero", func(t *testing.T) {
		t.Parallel()
		mn, mx, avg := seriesStats(nil, 10)
		require.Zero(t, mn)
		require.Zero(t, mx)
		require.Zero(t, avg)
	})

	t.Run("min max avg over the full series", func(t *testing.T) {
		t.Parallel()
		mn, mx, avg := seriesStats([]float64{2, 8, 5, 1}, 10)
		require.Equal(t, 1.0, mn)
		require.Equal(t, 8.0, mx)
		require.Equal(t, 4.0, avg)
	})

	t.Run("stats only cover the trailing window", func(t *testing.T) {
		t.Parallel()
		mn, mx, avg := seriesStats([]float64{100, 2, 4}, 2)
		require.Equal(t, 2.0, mn)
		require.Equal(t, 4.0, mx)
		require.Equal(t, 3.0, avg)
	})
}
