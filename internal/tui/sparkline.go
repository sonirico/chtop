package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// brailleBase is the Unicode code point of the empty braille cell (U+2800).
// A braille cell is 2 columns x 4 rows, so each cell packs 2 samples wide and
// 4 levels tall.
const brailleBase = 0x2800

// brailleLeft / brailleRight map a 0..4 fill height to the dot bits of the
// left / right sub-column, filled from the bottom up.
var (
	brailleLeft  = [5]rune{0, 0x40, 0x44, 0x46, 0x47}
	brailleRight = [5]rune{0, 0x80, 0xA0, 0xB0, 0xB8}
)

// brailleArea renders values as a filled area chart h character-rows tall and
// w columns wide using braille dots. Each cell is 2 samples wide and 4 dot
// levels tall, so the chart resolves 2*w samples horizontally and 4*h levels
// vertically. Values are normalised against the window peak and the most
// recent samples are anchored to the right edge. Returns h rows, each w runes
// wide (spaces where empty).
func brailleArea(values []float64, w, h int) []string {
	rows := make([]string, h)
	blank := strings.Repeat(" ", maxInt(w, 0))
	for i := range rows {
		rows[i] = blank
	}
	if w <= 0 || h <= 0 || len(values) == 0 {
		return rows
	}
	tail := values
	if samples := w * 2; len(values) > samples {
		tail = values[len(values)-samples:]
	}
	var peak float64
	for _, v := range tail {
		if v > peak {
			peak = v
		}
	}
	total := 4 * h
	fill := func(v float64) int {
		if peak == 0 {
			return 0
		}
		if v < 0 {
			v = 0
		}
		f := int(v/peak*float64(total) + 0.5)
		if f > total {
			f = total
		}
		return f
	}
	cells := (len(tail) + 1) / 2
	offset := w - cells // right-align so the newest sample hugs the right edge
	for r := range rows {
		buf := make([]rune, w)
		for c := range buf {
			buf[c] = ' '
		}
		base := (h - 1 - r) * 4 // dot level at the bottom of this char row
		for cell := 0; cell < cells; cell++ {
			lh := clampInt(fill(tail[2*cell])-base, 0, 4)
			rh := 0
			if 2*cell+1 < len(tail) {
				rh = clampInt(fill(tail[2*cell+1])-base, 0, 4)
			}
			if lh == 0 && rh == 0 {
				continue
			}
			buf[offset+cell] = rune(brailleBase) | brailleLeft[lh] | brailleRight[rh]
		}
		rows[r] = string(buf)
	}
	return rows
}

// brailleAreaStyled renders a braille area chart with a vertical colour
// gradient: bottom rows green, middle yellow, top red, so tall spikes read as
// hot (btop-style).
func brailleAreaStyled(values []float64, w, h int) []string {
	raw := brailleArea(values, w, h)
	for r := range raw {
		raw[r] = areaRowStyle(r, h).Render(raw[r])
	}
	return raw
}

// areaRowStyle picks the gradient colour for char row r of an h-row graph,
// where row 0 is the top.
func areaRowStyle(r, h int) lipgloss.Style {
	frac := float64(h-1-r) / float64(maxInt(h-1, 1)) // 0 at bottom, 1 at top
	switch {
	case frac >= 0.67:
		return sparkHigh
	case frac >= 0.34:
		return sparkMid
	default:
		return sparkLow
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// seriesStats returns the min, max and mean over the trailing `samples` values
// (the window the graph draws). Zero values on an empty slice.
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
