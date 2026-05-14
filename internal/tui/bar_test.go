package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// stripANSI is a crude but sufficient helper to drop lipgloss ANSI styling so
// the test asserts on visible characters only.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			// CSI sequence: skip until the final byte (an ASCII letter).
			for i < len(s) && !isAlpha(s[i]) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func TestRenderBar_VisibleChars(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name        string
		progress    float64
		width       int
		wantVisible string
	}
	fillColor := lipgloss.Color("9")
	emptyColor := lipgloss.Color("8")
	cases := []testCase{
		{"empty", 0.0, 10, "░░░░░░░░░░"},
		{"full", 1.0, 10, "██████████"},
		{"half", 0.5, 10, "█████░░░░░"},
		{"out of range high clamps to full", 1.5, 4, "████"},
		{"out of range low clamps to empty", -0.2, 4, "░░░░"},
		{"zero width returns empty string", 0.5, 0, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := stripANSI(renderBar(tc.progress, tc.width, fillColor, emptyColor))
			require.Equal(t, tc.wantVisible, got)
		})
	}
}

func TestRenderBar_WidthPreserved(t *testing.T) {
	t.Parallel()
	// Every non-zero width should produce exactly that many displayable cells
	// regardless of where the partial cell lands. Guards against off-by-one
	// drift inside the partial-cell logic.
	for w := 1; w <= 30; w++ {
		for _, p := range []float64{0.0, 0.13, 0.5, 0.66, 0.99, 1.0} {
			got := stripANSI(renderBar(p, w, lipgloss.Color("9"), lipgloss.Color("8")))
			require.Equal(
				t, w, runeCount(got),
				"width=%d progress=%.2f rendered=%q", w, p, got,
			)
		}
	}
}

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

func TestProgressColor_Boundaries(t *testing.T) {
	t.Parallel()
	// Smoke test: the function is a switch, just make sure each bucket is
	// reachable and we never panic on out-of-range values.
	require.NotNil(t, progressColor(0))
	require.NotNil(t, progressColor(0.32))
	require.NotNil(t, progressColor(0.33))
	require.NotNil(t, progressColor(0.74))
	require.NotNil(t, progressColor(0.75))
	require.NotNil(t, progressColor(1.0))
	require.NotNil(t, progressColor(-1))
	require.NotNil(t, progressColor(2))
}
