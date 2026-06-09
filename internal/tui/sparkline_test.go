package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestBrailleArea(t *testing.T) {
	t.Parallel()

	t.Run("returns h rows each w runes wide", func(t *testing.T) {
		t.Parallel()
		rows := brailleArea([]float64{1, 2, 3, 4, 5, 6}, 5, 3)
		require.Len(t, rows, 3)
		for _, r := range rows {
			require.Equal(t, 5, utf8.RuneCountInString(r))
		}
	})

	t.Run("no values gives h blank rows", func(t *testing.T) {
		t.Parallel()
		rows := brailleArea(nil, 4, 2)
		require.Len(t, rows, 2)
		for _, r := range rows {
			require.Equal(t, strings.Repeat(" ", 4), r)
		}
	})

	t.Run("zero width gives empty rows", func(t *testing.T) {
		t.Parallel()
		rows := brailleArea([]float64{1, 2, 3}, 0, 3)
		require.Len(t, rows, 3)
		for _, r := range rows {
			require.Equal(t, "", r)
		}
	})

	t.Run("peak sample fills the cell to full height", func(t *testing.T) {
		t.Parallel()
		// One max sample in a 1x1 graph fills the left sub-column to 4 dots.
		rows := brailleArea([]float64{1.0}, 1, 1)
		require.Len(t, rows, 1)
		require.Equal(t, "⡇", rows[0]) // left column full: dots 1-3-2-7
	})

	t.Run("zero values produce all spaces", func(t *testing.T) {
		t.Parallel()
		rows := brailleArea([]float64{0, 0, 0}, 3, 2)
		for _, r := range rows {
			require.Equal(t, strings.Repeat(" ", 3), r)
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
