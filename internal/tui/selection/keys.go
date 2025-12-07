package selection

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the selection UI
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Toggle       key.Binding
	SelectAll    key.Binding
	SelectNone   key.Binding
	Confirm      key.Binding
	Quit         key.Binding
	Search       key.Binding
	ClearSearch     key.Binding
	Ignore          key.Binding
	IgnoreCategory  key.Binding
	SaveProfile     key.Binding
	TabTap       key.Binding
	TabBrew      key.Binding
	TabCask      key.Binding
	TabVSCode    key.Binding
	TabCursor    key.Binding
	TabGo        key.Binding
	TabMas       key.Binding
	TabAll       key.Binding
	Help         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
}

// DefaultKeyMap returns the default keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "prev tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next tab"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all"),
		),
		SelectNone: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "select none"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		ClearSearch: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear search"),
		),
		Ignore: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "ignore selected"),
		),
		IgnoreCategory: key.NewBinding(
			key.WithKeys("I", "C"),
			key.WithHelp("I/C", "ignore category"),
		),
		SaveProfile: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save profile"),
		),
		TabTap: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "taps"),
		),
		TabBrew: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "brews"),
		),
		TabCask: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "casks"),
		),
		TabVSCode: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "vscode"),
		),
		TabCursor: key.NewBinding(
			key.WithKeys("5"),
			key.WithHelp("5", "cursor"),
		),
		TabGo: key.NewBinding(
			key.WithKeys("6"),
			key.WithHelp("6", "go"),
		),
		TabMas: key.NewBinding(
			key.WithKeys("7"),
			key.WithHelp("7", "mas"),
		),
		TabAll: key.NewBinding(
			key.WithKeys("0"),
			key.WithHelp("0", "all"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
	}
}

// ShortHelp returns keybindings shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.SelectAll, k.SelectNone, k.Confirm, k.Quit, k.Search}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.PageUp, k.PageDown},
		{k.Toggle, k.SelectAll, k.SelectNone, k.Ignore, k.IgnoreCategory},
		{k.TabAll, k.TabTap, k.TabBrew, k.TabCask},
		{k.TabVSCode, k.TabCursor, k.TabGo, k.TabMas},
		{k.Search, k.Confirm, k.Quit, k.Help},
	}
}
