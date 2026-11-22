package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

type messageKind int

const (
	messageInfo messageKind = iota
	messageWarning
	messageAlert
)

// StatusBarModel manages the bottom status bar.
type StatusBarModel struct {
	Width    int
	ReadOnly bool   // show [READ-ONLY] badge
	Message  string // transient message (e.g. "Saved!")
	kind     messageKind
	msgGen   int
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

func (m *StatusBarModel) setMessage(msg string, kind messageKind) {
	m.Message = flattenText(msg)
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
	var left string
	if m.Message != "" {
		left = m.messageStyle(theme).Render(m.Message)
	} else {
		diffStats := ""
		if len(stats) > 0 {
			diffStats = formatDiffStats(stats[0], theme)
		}
		left = modeHints(theme, keys, mode, focus, diffStats)
	}

	textWidth := theme.barRoom(m.Width)
	right := ""
	if m.ReadOnly {
		right = theme.EmptyWarning.Render("[READ-ONLY]")
	}
	if fileName != "" {
		if right != "" {
			right += "  "
		}
		count := fmt.Sprintf("  %d vars", varCount)
		name := fitName(fileName, textWidth-lipgloss.Width(right)-lipgloss.Width(count))
		right += theme.MutedItem.Render(name + count)
	}

	return theme.StatusBar.
		Width(m.Width).
		Render(statusRows(left, right, textWidth))
}

func statusRows(left, right string, textWidth int) string {
	rows := wrapRows(left, textWidth)
	if right != "" {
		room := textWidth - lipgloss.Width(right)
		last := rows[len(rows)-1]
		gap := room - lipgloss.Width(last)
		if last != "" && gap <= 0 {
			rows = append(rows, "")
			last, gap = "", room
		}
		rows[len(rows)-1] = last + strings.Repeat(" ", max(gap, 0)) + right
	}
	return strings.Join(rows, "\n")
}

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
		parts = append(parts, theme.DiffChanged.Render(fmt.Sprintf("%d%s", s.Changed, diffChangedGlyph)))
	}
	if s.Added > 0 {
		parts = append(parts, theme.DiffAdded.Render(fmt.Sprintf("%d%s", s.Added, diffAddedGlyph)))
	}
	if s.Removed > 0 {
		parts = append(parts, theme.DiffRemoved.Render(fmt.Sprintf("%d%s", s.Removed, diffRemovedGlyph)))
	}
	if len(parts) == 0 {
		return theme.DiffEqual.Render("all equal")
	}
	return strings.Join(parts, " ")
}

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
			formatHint(theme, "vars", keys.Enter),
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
		return ""
	case ModeHelp:
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

func formatHint(theme Theme, desc string, bindings ...key.Binding) string {
	return fmt.Sprintf("[%s]%s", theme.StatusBarKey.Render(shortKeys(bindings...)), theme.MutedItem.Render(desc))
}

func inputHint(theme Theme, keys KeyMap, action string) string {
	return theme.MutedItem.Render(fmt.Sprintf("%s to %s, %s to cancel", shortKeys(keys.Enter), action, shortKeys(keys.Escape)))
}
