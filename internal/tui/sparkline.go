package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// brailleBase is the Unicode code point of the empty braille cell (U+2800).
// A braille cell is 2 columns x 4 rows, so two samples pack into one rune.
const brailleBase = 0x2800

// brailleLeft / brailleRight map a 0..4 fill height to the dot bits of the
// left / right sub-column, filled from the bottom up.
var (
	brailleLeft  = [5]rune{0, 0x40, 0x44, 0x46, 0x47}
	brailleRight = [5]rune{0, 0x80, 0xA0, 0xB0, 0xB8}
)

// sparklineColored renders the trailing 2*width samples as a braille bar chart
// coloured by intensity: low values stay green, mid yellow, hot values red.
// Two samples share one cell (double the horizontal resolution of the block
// ramp); each cell is coloured by the taller of its two bars. Returns the
// empty string when there is nothing to plot.
func sparklineColored(values []float64, width int) string {
	if width <= 0 || len(values) == 0 {
		return ""
	}
	tail := values
	if samples := width * 2; len(values) > samples {
		tail = values[len(values)-samples:]
	}
	var max float64
	for _, v := range tail {
		if v > max {
			max = v
		}
	}
	height := func(v float64) int {
		if max == 0 {
			return 1 // faint baseline so a flat/empty series still shows
		}
		if v < 0 {
			v = 0
		}
		h := int(v/max*4 + 0.5)
		if h < 0 {
			h = 0
		}
		if h > 4 {
			h = 4
		}
		return h
	}
	bands := [3]lipgloss.Style{sparkLow, sparkMid, sparkHigh}
	bandOf := func(h int) int {
		switch {
		case h >= 4:
			return 2
		case h >= 3:
			return 1
		default:
			return 0
		}
	}

	var b, run strings.Builder
	curBand := -1
	flush := func() {
		if run.Len() == 0 {
			return
		}
		b.WriteString(bands[curBand].Render(run.String()))
		run.Reset()
	}
	for i := 0; i < len(tail); i += 2 {
		lh := height(tail[i])
		rh := 0
		if i+1 < len(tail) {
			rh = height(tail[i+1])
		}
		peak := lh
		if rh > peak {
			peak = rh
		}
		if band := bandOf(peak); band != curBand {
			flush()
			curBand = band
		}
		run.WriteRune(rune(brailleBase) | brailleLeft[lh] | brailleRight[rh])
	}
	flush()
	return b.String()
}

// seriesStats returns the min, max and mean over the trailing `samples` values
// (the window the sparkline draws). Zero values on an empty slice.
func seriesStats(values []float64, samples int) (mn, mx, avg float64) {
	tail := values
	if samples > 0 && len(values) > samples {
		tail = values[len(values)-samples:]
	}
	if len(tail) == 0 {
		return
	}
	mn = tail[0]
	var sum float64
	for i, v := range tail {
		if i == 0 || v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
		sum += v
	}
	return mn, mx, sum / float64(len(tail))
}
