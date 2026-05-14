package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// filter is a tiny `/`-driven substring filter, ported verbatim from
// readpanda. See the comments there for the contract.
type filter struct {
	active bool
	query  string
}

func (f *filter) Handle(k tea.KeyMsg) (consumed, applied bool) {
	if !f.active {
		if k.String() == "/" {
			f.active = true
			f.query = ""
			return true, true
		}
		return false, false
	}
	switch k.String() {
	case "esc":
		f.active = false
		f.query = ""
		return true, true
	case "enter":
		f.active = false
		return true, false
	case "backspace":
		if len(f.query) > 0 {
			f.query = f.query[:len(f.query)-1]
			return true, true
		}
		return true, false
	}
	if len(k.String()) == 1 {
		f.query += k.String()
		return true, true
	}
	return true, false
}

func (f *filter) Match(name string) bool {
	if f.query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(f.query))
}

func (f *filter) Bar() string {
	if !f.active && f.query == "" {
		return ""
	}
	prefix := "/"
	if !f.active {
		prefix = "[filter] "
	}
	return prefix + f.query
}
