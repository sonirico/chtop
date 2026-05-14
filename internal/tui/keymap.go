package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap declares every key binding the app reacts to.
type KeyMap struct {
	Quit    key.Binding
	Back    key.Binding
	Help    key.Binding
	Refresh key.Binding
	Command key.Binding
	Enter   key.Binding
	Kill    key.Binding
}

func newKeyMap() KeyMap {
	return KeyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Back:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Command: key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Kill:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "kill query")),
	}
}
