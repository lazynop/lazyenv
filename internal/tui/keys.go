package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
)

// KeyMap defines all keybindings for the app.
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Left           key.Binding
	Right          key.Binding
	Enter          key.Binding
	Edit           key.Binding
	EditKey        key.Binding
	EditRight      key.Binding
	Add            key.Binding
	Delete         key.Binding
	Compare        key.Binding
	Search         key.Binding
	Save           key.Binding
	Reset          key.Binding
	ToggleSecret   key.Binding
	ToggleSort     key.Binding
	ToggleGroup    key.Binding
	ToggleCollapse key.Binding
	Help           key.Binding
	Quit           key.Binding
	Escape         key.Binding
	Confirm        key.Binding
	Deny           key.Binding
	NextDiff       key.Binding
	PrevDiff       key.Binding
	Filter         key.Binding
	Matrix         key.Binding
	YankValue      key.Binding
	YankLine       key.Binding
	Peek           key.Binding
	CreateFile     key.Binding
	DuplicateFile  key.Binding
	DeleteFile     key.Binding
	RenameFile     key.Binding
	TemplateFile   key.Binding
	Reorder        key.Binding
	ReorderAlpha   key.Binding
	ReorderGrouped key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	Home           key.Binding
	End            key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:             key.NewBinding(key.WithKeys("up", "k")),
		Down:           key.NewBinding(key.WithKeys("down", "j")),
		Left:           key.NewBinding(key.WithKeys("left", "h")),
		Right:          key.NewBinding(key.WithKeys("right", "l")),
		Enter:          key.NewBinding(key.WithKeys("enter")),
		Edit:           key.NewBinding(key.WithKeys("e")),
		EditKey:        key.NewBinding(key.WithKeys("E")),
		EditRight:      key.NewBinding(key.WithKeys("E")),
		Add:            key.NewBinding(key.WithKeys("a")),
		Delete:         key.NewBinding(key.WithKeys("d")),
		Compare:        key.NewBinding(key.WithKeys("c")),
		Search:         key.NewBinding(key.WithKeys("/")),
		Save:           key.NewBinding(key.WithKeys("w")),
		Reset:          key.NewBinding(key.WithKeys("r")),
		ToggleSecret:   key.NewBinding(key.WithKeys("ctrl+s")),
		ToggleSort:     key.NewBinding(key.WithKeys("o")),
		ToggleGroup:    key.NewBinding(key.WithKeys("g")),
		ToggleCollapse: key.NewBinding(key.WithKeys("space")),
		Help:           key.NewBinding(key.WithKeys("?")),
		Quit:           key.NewBinding(key.WithKeys("q", "ctrl+c")),
		Escape:         key.NewBinding(key.WithKeys("esc")),
		Confirm:        key.NewBinding(key.WithKeys("y")),
		Deny:           key.NewBinding(key.WithKeys("n")),
		NextDiff:       key.NewBinding(key.WithKeys("n")),
		PrevDiff:       key.NewBinding(key.WithKeys("N")),
		Filter:         key.NewBinding(key.WithKeys("f")),
		Matrix:         key.NewBinding(key.WithKeys("m")),
		YankValue:      key.NewBinding(key.WithKeys("y")),
		YankLine:       key.NewBinding(key.WithKeys("Y")),
		Peek:           key.NewBinding(key.WithKeys("p")),
		CreateFile:     key.NewBinding(key.WithKeys("N")),
		DuplicateFile:  key.NewBinding(key.WithKeys("C")),
		DeleteFile:     key.NewBinding(key.WithKeys("D")),
		RenameFile:     key.NewBinding(key.WithKeys("R")),
		TemplateFile:   key.NewBinding(key.WithKeys("T")),
		Reorder:        key.NewBinding(key.WithKeys("O")),
		ReorderAlpha:   key.NewBinding(key.WithKeys("a")),
		ReorderGrouped: key.NewBinding(key.WithKeys("g")),
		PageUp:         key.NewBinding(key.WithKeys("pgup")),
		PageDown:       key.NewBinding(key.WithKeys("pgdown")),
		Home:           key.NewBinding(key.WithKeys("home")),
		End:            key.NewBinding(key.WithKeys("end")),
	}
}

// keyLabels are the labels of the key names that are not a single character
// or a ctrl+<x> combination.
var keyLabels = map[string]string{
	"up":     "↑",
	"down":   "↓",
	"left":   "←",
	"right":  "→",
	"enter":  "Enter",
	"esc":    "Esc",
	"space":  "Space",
	"pgup":   "PgUp",
	"pgdown": "PgDn",
	"home":   "Home",
	"end":    "End",
}

// KeyLabel returns how the help screen, the status-bar hints, the prompts and
// docs/keybindings.md write the key named k: a single character as it is,
// ctrl+<x> as Ctrl+<X>, any other name from keyLabels, or as it is when
// keyLabels has no entry for it.
func KeyLabel(k string) string {
	label, _ := keyLabel(k)
	return label
}

// keyLabel is KeyLabel, also reporting whether k has a label of its own.
func keyLabel(k string) (string, bool) {
	if len([]rune(k)) == 1 {
		return k, true
	}
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return "Ctrl+" + strings.ToUpper(c), true
	}
	if label, ok := keyLabels[k]; ok {
		return label, true
	}
	return k, false
}
