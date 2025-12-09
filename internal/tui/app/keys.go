package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the main TUI
type KeyMap struct {
	// Global keys
	Quit key.Binding
	Help key.Binding
}

// DefaultKeyMap returns the default keybindings for the main TUI
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns keybindings shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Help}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Help},
	}
}
