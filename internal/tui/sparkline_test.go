package tui

import (
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
