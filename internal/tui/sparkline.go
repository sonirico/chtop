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

// blockFills maps a sub-row fill height (0..8) to a unicode block char.
var blockFills = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// areaGraph renders values as a filled area chart h rows tall and w columns wide.
// Values are normalized against their window max. Returns h rows of text, each
// w runes wide, using unicode block elements (U+2581..U+2588).
func areaGraph(values []float64, w, h int) []string {
	rows := make([]string, h)
	empty := strings.Repeat(" ", maxInt(w, 0))
	for i := range rows {
		rows[i] = empty
	}
	if w <= 0 || h <= 0 || len(values) == 0 {
		return rows
	}
	tail := values
	if len(values) > w {
		tail = values[len(values)-w:]
	}
	var peak float64
	for _, v := range tail {
		if v > peak {
			peak = v
		}
	}
	// colFill[c] is fill in sub-row units (0..h*8).
	colFill := make([]int, w)
	offset := w - len(tail)
	for i, v := range tail {
		col := offset + i
		if col < 0 || peak == 0 {
			continue
		}
		f := int(v / peak * float64(h*8))
		if f > h*8 {
			f = h * 8
		}
		colFill[col] = f
	}
	// Build rows top (0) to bottom (h-1).
	bufs := make([][]rune, h)
	for r := range bufs {
		bufs[r] = make([]rune, w)
		for c := range bufs[r] {
			bufs[r][c] = ' '
		}
	}
	for col, fill := range colFill {
		for row := 0; row < h; row++ {
			tierBot := (h - 1 - row) * 8
			switch {
			case fill >= tierBot+8:
				bufs[row][col] = '█'
			case fill > tierBot:
				bufs[row][col] = blockFills[fill-tierBot]
			}
		}
	}
	for r, buf := range bufs {
		rows[r] = string(buf)
	}
	return rows
}

// areaGraphStyled renders an area graph and applies colorPrimary to every row.
func areaGraphStyled(values []float64, w, h int) []string {
	raw := areaGraph(values, w, h)
	st := lipgloss.NewStyle().Foreground(colorPrimary)
	for i, line := range raw {
		raw[i] = st.Render(line)
	}
	return raw
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
