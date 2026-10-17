package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// messageKind picks the theme color a status-bar message renders in.
type messageKind int

const (
	messageInfo    messageKind = iota // primary color
	messageWarning                    // warning color: an action refused because of the input or the current state
	messageAlert                      // error color: a failure, or a prompt confirming a delete
)

// StatusBarModel manages the bottom status bar.
type StatusBarModel struct {
	Width    int
	ReadOnly bool        // show [READ-ONLY] badge
	Message  string      // transient message (e.g. "Saved!")
	kind     messageKind // color Message renders in
	Mode     string      // current mode label
	msgGen   int         // bumped on every message set; guards against stale clears
}

// NewStatusBarModel creates a new status bar model.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{}
}

// SetMessage sets a status message, rendered in the primary color.
func (m *StatusBarModel) SetMessage(msg string) {
	m.setMessage(msg, messageInfo)
}

// SetWarning sets a status message reporting an action refused because of the
// input or the current state, rendered in the warning color.
func (m *StatusBarModel) SetWarning(msg string) {
	m.setMessage(msg, messageWarning)
}

// SetAlert sets a status message that reports a failure or asks to confirm a
// delete, rendered in the error color.
func (m *StatusBarModel) SetAlert(msg string) {
	m.setMessage(msg, messageAlert)
}

// setMessage sets the message and its kind. Each call bumps a generation
// counter so a clear scheduled for an earlier message cannot wipe a newer one.
// This keeps persistent prompts (menus, confirmations) on screen even if a
// previous transient flash's auto-clear timer fires while they are showing.
func (m *StatusBarModel) setMessage(msg string, kind messageKind) {
	m.Message = msg
	m.kind = kind
	m.msgGen++
}

// ClearMessage clears the transient message.
func (m *StatusBarModel) ClearMessage() {
	m.Message = ""
	m.kind = messageInfo
}

// View renders the status bar, its hints naming the keys that keys binds.
func (m *StatusBarModel) View(theme Theme, keys KeyMap, mode AppMode, focus Focus, fileName string, varCount int, stats ...DiffStats) string {
	// Build diff stats string if provided
	diffStats := ""
	if len(stats) > 0 {
		diffStats = formatDiffStats(stats[0], theme)
	}

	// Left side: keybinding hints based on mode and focus
	hints := modeHints(theme, keys, mode, focus, diffStats)

	// Right side: file info + read-only badge
	right := ""
	if m.ReadOnly {
		right = theme.EmptyWarning.Render("[READ-ONLY]")
	}
	if fileName != "" {
		fileInfo := theme.MutedItem.Render(fmt.Sprintf("%s  %d vars", fileName, varCount))
		if right != "" {
			right = right + "  " + fileInfo
		} else {
			right = fileInfo
		}
	}

	// Transient message overrides hints
	left := hints
	if m.Message != "" {
		left = m.messageStyle(theme).Render(m.Message)
	}

	// Layout
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := max(m.Width-leftWidth-rightWidth-4, 1)

	bar := fmt.Sprintf("  %s%s%s  ", left, strings.Repeat(" ", gap), right)

	return theme.StatusBar.
		Width(m.Width).
		Render(bar)
}

// messageStyle returns the style Message renders in, picked by its kind.
func (m *StatusBarModel) messageStyle(theme Theme) lipgloss.Style {
	switch m.kind {
	case messageWarning:
		return theme.StatusBarWarning
	case messageAlert:
		return theme.StatusBarAlert
	default:
		return theme.SelectedItem
	}
}

func formatDiffStats(s DiffStats, theme Theme) string {
	var parts []string
	if s.Changed > 0 {
		parts = append(parts, theme.DiffChanged.Render(fmt.Sprintf("%d≠", s.Changed)))
	}
	if s.Added > 0 {
		parts = append(parts, theme.DiffAdded.Render(fmt.Sprintf("%d+", s.Added)))
	}
	if s.Removed > 0 {
		parts = append(parts, theme.DiffRemoved.Render(fmt.Sprintf("%d-", s.Removed)))
	}
	if len(parts) == 0 {
		return theme.DiffEqual.Render("all equal")
	}
	return strings.Join(parts, " ")
}

// modeHints is the status-bar hint of mode and focus, naming the keys that
// keys binds; diffStats ends the compare view's hint.
func modeHints(theme Theme, keys KeyMap, mode AppMode, focus Focus, diffStats string) string {
	switch mode {
	case ModeNormal:
		if focus == FocusVars {
			return strings.Join([]string{
				formatHint(theme, "edit", keys.Edit),
				formatHint(theme, "rename", keys.EditKey),
				formatHint(theme, "add", keys.Add),
				formatHint(theme, "del", keys.Delete),
				formatHint(theme, "yank", keys.YankValue),
				formatHint(theme, "peek", keys.Peek),
				formatHint(theme, "search", keys.Search),
				formatHint(theme, "sort", keys.ToggleSort),
				formatHint(theme, "group", keys.ToggleGroup),
				formatHint(theme, "save", keys.Save),
				formatHint(theme, "reset", keys.Reset),
				formatHint(theme, "quit", keys.Quit),
			}, " ")
		}
		return strings.Join([]string{
			formatHint(theme, "select", keys.Enter),
			formatHint(theme, "compare", keys.Compare),
			formatHint(theme, "matrix", keys.Matrix),
			formatHint(theme, "sort", keys.ToggleSort),
			formatHint(theme, "group", keys.ToggleGroup),
			formatHint(theme, "save", keys.Save),
			formatHint(theme, "reset", keys.Reset),
			formatHint(theme, "help", keys.Help),
			formatHint(theme, "quit", keys.Quit),
		}, " ")
	case ModeCompareSelect:
		return theme.EmptyWarning.Render(fmt.Sprintf("Select second file to compare (%s to pick, %s to cancel)",
			shortKeys(keys.Enter), shortKeys(keys.Escape, keys.Quit)))
	case ModeComparing:
		return strings.Join([]string{
			formatHint(theme, "copy", keys.Left, keys.Right),
			formatHint(theme, "edit L/R", keys.Edit, keys.EditRight),
			formatHint(theme, "next/prev diff", keys.NextDiff, keys.PrevDiff),
			formatHint(theme, "filter", keys.Filter),
			formatHint(theme, "secrets", keys.ToggleSecret),
			formatHint(theme, "save", keys.Save),
			formatHint(theme, "reset", keys.Reset),
			formatHint(theme, "back", keys.Quit),
			theme.MutedItem.Render("│"),
			diffStats,
		}, " ")
	case ModeEditing:
		return inputHint(theme, keys, "confirm")
	case ModeConfirmDelete:
		return theme.DuplicateWarn.Render("Delete variable? " + yesNo(keys))
	case ModeConfirmDeleteFile, ModeConfirmMatrixDelete:
		// The prompt set on entering the mode names what gets deleted and
		// stays until the mode ends, so a hint here would never show.
		return ""
	case ModeHelp:
		// The help screen fills the terminal with no status bar; its own
		// footer names the keys.
		return ""
	case ModeSearching:
		return theme.MutedItem.Render(fmt.Sprintf("Type to search, %s to keep the filter, %s to clear it",
			shortKeys(keys.Enter), shortKeys(keys.Escape)))
	case ModeCreateFile:
		return inputHint(theme, keys, "create")
	case ModeDuplicateFile:
		return inputHint(theme, keys, "duplicate")
	case ModeRenameFile:
		return inputHint(theme, keys, "rename")
	case ModeTemplateFile:
		return inputHint(theme, keys, "create template")
	case ModeMatrix:
		return strings.Join([]string{
			formatHint(theme, "navigate", keys.Up, keys.Down, keys.Left, keys.Right),
			formatHint(theme, "add missing", keys.Add),
			formatHint(theme, "delete", keys.Delete),
			formatHint(theme, "sort", keys.ToggleSort),
			formatHint(theme, "back", keys.Quit),
		}, " ")
	default:
		return ""
	}
}

// formatHint is the hint "[key]desc" naming the short keys of bindings.
func formatHint(theme Theme, desc string, bindings ...key.Binding) string {
	return fmt.Sprintf("[%s]%s", theme.StatusBarKey.Render(shortKeys(bindings...)), theme.MutedItem.Render(desc))
}

// inputHint is the hint of an input bar: the key that confirms it, to do
// action, and the one that cancels it.
func inputHint(theme Theme, keys KeyMap, action string) string {
	return theme.MutedItem.Render(fmt.Sprintf("%s to %s, %s to cancel", shortKeys(keys.Enter), action, shortKeys(keys.Escape)))
}
