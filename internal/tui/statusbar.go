package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// StatusBarModel manages the bottom status bar.
type StatusBarModel struct {
	Width    int
	ReadOnly bool   // show [READ-ONLY] badge
	Message  string // transient message (e.g. "Saved!")
	Mode     string // current mode label
}

// NewStatusBarModel creates a new status bar model.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{}
}

// SetMessage sets a transient status message.
func (m *StatusBarModel) SetMessage(msg string) {
	m.Message = msg
}

// ClearMessage clears the transient message.
func (m *StatusBarModel) ClearMessage() {
	m.Message = ""
}

// View renders the status bar.
func (m *StatusBarModel) View(theme Theme, mode AppMode, focus Focus, fileName string, varCount int, stats ...DiffStats) string {
	// Build diff stats string if provided
	diffStats := ""
	if len(stats) > 0 {
		diffStats = formatDiffStats(stats[0], theme)
	}

	// Left side: keybinding hints based on mode and focus
	hints := modeHints(theme, mode, focus, diffStats)

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
		left = theme.SelectedItem.Render(m.Message)
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

func modeHints(theme Theme, mode AppMode, focus Focus, diffStats string) string {
	switch mode {
	case ModeNormal:
		if focus == FocusVars {
			return fmt.Sprintf(
				"%s %s %s %s %s %s %s %s %s %s %s %s",
				formatHint(theme, "e", "edit"),
				formatHint(theme, "E", "rename"),
				formatHint(theme, "a", "add"),
				formatHint(theme, "d", "del"),
				formatHint(theme, "y", "yank"),
				formatHint(theme, "p", "peek"),
				formatHint(theme, "/", "search"),
				formatHint(theme, "o", "sort"),
				formatHint(theme, "g", "group"),
				formatHint(theme, "w", "save"),
				formatHint(theme, "r", "reset"),
				formatHint(theme, "q", "quit"),
			)
		}
		return fmt.Sprintf(
			"%s %s %s %s %s %s %s %s %s",
			formatHint(theme, "enter", "select"),
			formatHint(theme, "c", "compare"),
			formatHint(theme, "m", "matrix"),
			formatHint(theme, "o", "sort"),
			formatHint(theme, "g", "group"),
			formatHint(theme, "w", "save"),
			formatHint(theme, "r", "reset"),
			formatHint(theme, "?", "help"),
			formatHint(theme, "q", "quit"),
		)
	case ModeCompareSelect:
		return theme.EmptyWarning.Render("Select second file to compare (Enter to select, Esc to cancel)")
	case ModeComparing:
		return fmt.Sprintf(
			"%s %s %s %s %s %s %s %s %s %s",
			formatHint(theme, "←/→", "copy"),
			formatHint(theme, "e/E", "edit L/R"),
			formatHint(theme, "n/N", "next/prev diff"),
			formatHint(theme, "f", "filter"),
			formatHint(theme, "^S", "secrets"),
			formatHint(theme, "w", "save"),
			formatHint(theme, "r", "reset"),
			formatHint(theme, "q", "back"),
			theme.MutedItem.Render("│"),
			diffStats,
		)
	case ModeEditing:
		return theme.MutedItem.Render("Enter to confirm, Esc to cancel")
	case ModeConfirmDelete, ModeConfirmMatrixDelete:
		return theme.DuplicateWarn.Render("Delete variable? (y/n)")
	case ModeHelp:
		return theme.MutedItem.Render("Press Esc or ? to close help")
	case ModeSearching:
		return theme.MutedItem.Render("Type to search, Esc to close")
	case ModeCreateFile:
		return theme.MutedItem.Render("Enter to create, Esc to cancel")
	case ModeDuplicateFile:
		return theme.MutedItem.Render("Enter to duplicate, Esc to cancel")
	case ModeConfirmDeleteFile:
		return theme.DuplicateWarn.Render("Delete file from disk? (y/n)")
	case ModeRenameFile:
		return theme.MutedItem.Render("Enter to rename, Esc to cancel")
	case ModeTemplateFile:
		return theme.MutedItem.Render("Enter to create template, Esc to cancel")
	case ModeMatrix:
		return fmt.Sprintf(
			"%s %s %s %s %s",
			formatHint(theme, "↑↓←→", "navigate"),
			formatHint(theme, "a", "add missing"),
			formatHint(theme, "d", "delete"),
			formatHint(theme, "o", "sort"),
			formatHint(theme, "q", "back"),
		)
	default:
		return ""
	}
}

func formatHint(theme Theme, key, desc string) string {
	return fmt.Sprintf("[%s]%s", theme.StatusBarKey.Render(key), theme.MutedItem.Render(desc))
}
