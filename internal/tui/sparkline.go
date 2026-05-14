package tui

import "strings"

// sparkBlocks is the 8-level vertical block ramp.
const sparkBlocks = "▁▂▃▄▅▆▇█"

// sparkline renders the trailing `width` samples of values as a one-character
// per sample unicode bar chart. Pure: same inputs always yield the same
// output. Returns the empty string when there is nothing to plot or width
// is non-positive.
func sparkline(values []float64, width int) string {
	if width <= 0 || len(values) == 0 {
		return ""
	}
	start := 0
	if len(values) > width {
		start = len(values) - width
	}
	tail := values[start:]
	var max float64
	for _, v := range tail {
		if v > max {
			max = v
		}
	}
	if max == 0 {
		// Render the baseline so empty/flat series still leave a visual gap.
		return strings.Repeat("▁", len(tail))
	}
	blocks := []rune(sparkBlocks)
	var b strings.Builder
	for _, v := range tail {
		if v < 0 {
			v = 0
		}
		// 8 buckets indexed 0..7. Use floor (truncate) so values land in the
		// expected block — adding 0.5 before truncating biases each bucket
		// upward and disagrees with the table-driven tests' expectations.
		idx := int(v / max * 7)
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}
