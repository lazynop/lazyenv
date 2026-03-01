package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines all keybindings for the app.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Enter        key.Binding
	Edit         key.Binding
	Add          key.Binding
	Delete       key.Binding
	Compare      key.Binding
	Search       key.Binding
	Save         key.Binding
	Reset        key.Binding
	ToggleSecret key.Binding
	ToggleSort   key.Binding
	Help         key.Binding
	Quit         key.Binding
	Escape       key.Binding
	Confirm      key.Binding
	Deny         key.Binding
	NextDiff     key.Binding
	PrevDiff     key.Binding
	Filter       key.Binding
	Matrix       key.Binding
	YankValue    key.Binding
	YankLine     key.Binding
	Peek         key.Binding
}

// DefaultKeyMap returns the default keybindings.
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
			key.WithHelp("←/h", "files"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "vars"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Compare: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "compare"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Save: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "save"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset"),
		),
		ToggleSecret: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "toggle secrets"),
		),
		ToggleSort: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "sort"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		Deny: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
		NextDiff: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next diff"),
		),
		PrevDiff: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev diff"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
		Matrix: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "matrix"),
		),
		YankValue: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank value"),
		),
		YankLine: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "yank KEY=value"),
		),
		Peek: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "peek original"),
		),
	}
}
