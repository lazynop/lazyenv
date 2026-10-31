package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/config"
)

func TestStatusBarSetClearMessage(t *testing.T) {
	sb := NewStatusBarModel()
	assert.Equal(t, "", sb.Message)

	sb.SetMessage("Saved!")
	assert.Equal(t, "Saved!", sb.Message)

	sb.ClearMessage()
	assert.Equal(t, "", sb.Message)
}

// TestStatusBarMessagesAreFlattened sets messages naming a file whose name
// holds a line break and an escape sequence, as a name can hold any byte but /
// and NUL: each kind of message keeps one row and sends the terminal no
// sequence.
func TestStatusBarMessagesAreFlattened(t *testing.T) {
	for name, set := range map[string]func(*StatusBarModel, string){
		"message": (*StatusBarModel).SetMessage,
		"warning": (*StatusBarModel).SetWarning,
		"alert":   (*StatusBarModel).SetAlert,
	} {
		sb := NewStatusBarModel()
		set(&sb, "Saved .env\n\x1b]0;x\x07")
		assert.Equal(t, "Saved .env↵␛]0;x␇", sb.Message, name)
	}
}

func TestStatusBarViewNormal(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	// Normal mode, file panel
	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "compare")
	assert.Contains(t, view, "help")
	assert.Contains(t, view, ".env")
	assert.Contains(t, view, "5 vars")
}

func TestStatusBarViewNormalVarFocus(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusVars, ".env", 3)
	assert.Contains(t, view, "edit")
	assert.Contains(t, view, "add")
	assert.Contains(t, view, "del")
	assert.Contains(t, view, "yank")
}

func TestStatusBarViewComparing(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	stats := DiffStats{Changed: 2, Added: 1, Removed: 0, Equal: 3}
	view := sb.View(theme, DefaultKeyMap(), ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "copy")
	assert.Contains(t, view, "filter")
}

func TestStatusBarViewComparingAllEqual(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	stats := DiffStats{Equal: 5}
	view := sb.View(theme, DefaultKeyMap(), ModeComparing, FocusVars, "", 0, stats)
	assert.Contains(t, view, "all equal")
}

func TestStatusBarViewEditing(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeEditing, FocusVars, "", 0)
	assert.Contains(t, view, "Enter to confirm")
}

func TestStatusBarViewConfirmDelete(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeConfirmDelete, FocusVars, "", 0)
	assert.Contains(t, view, "Delete variable")
}

func TestStatusBarViewSearching(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeSearching, FocusVars, "", 0)
	assert.Contains(t, view, "Type to search")
}

func TestStatusBarViewMatrix(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeMatrix, FocusVars, "", 0)
	assert.Contains(t, view, "navigate")
	assert.Contains(t, view, "add missing")
}

func TestStatusBarViewCompareSelect(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeCompareSelect, FocusFiles, "", 0)
	assert.Contains(t, view, "Select second file")
}

func TestStatusBarReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.ReadOnly = true
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.Contains(t, view, "READ-ONLY")
	assert.Contains(t, view, ".env")
}

// TestStatusBarKeepsTheFileInfoWholeOnTheLastRow renders the bar over a file
// at every width from the minimum up to one where it fits a row, with either
// panel focused, read-only or not. Wherever the bar wraps, the file info (the
// read-only badge, the file name and its variable count) stays whole on the
// last row, right-aligned as on a single row, and every row keeps the bar's
// left margin; the words read as on a single row, so no hint is lost.
func TestStatusBarKeepsTheFileInfoWholeOnTheLastRow(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})
	for _, focus := range []Focus{FocusFiles, FocusVars} {
		for _, readOnly := range []bool{false, true} {
			info := ".env.test  16 vars"
			if readOnly {
				info = "[READ-ONLY]  " + info
			}
			render := func(width int) []string {
				sb := NewStatusBarModel()
				sb.Width = width
				sb.ReadOnly = readOnly
				view := sb.View(theme, DefaultKeyMap(), ModeNormal, focus, ".env.test", 16)
				return strings.Split(stripAnsi(view), "\n")
			}
			words := strings.Fields(strings.Join(render(200), " "))
			for width := MinTerminalWidth; width <= 140; width++ {
				at := fmt.Sprintf("focus %d, read-only %v, width %d", focus, readOnly, width)
				rows := render(width)
				last := strings.TrimRight(rows[len(rows)-1], " ")
				assert.True(t, strings.HasSuffix(last, info), "%s: the last row %q does not end with %q", at, last, info)
				assert.Equal(t, width-1, lipgloss.Width(last), "%s: the file info is not right-aligned", at)
				for _, row := range rows {
					assert.True(t, strings.HasPrefix(row, "   "), "%s: row %q lost the left margin", at, row)
					assert.LessOrEqual(t, lipgloss.Width(row), width, "%s: row %q is wider than the bar", at, row)
				}
				assert.Equal(t, words, strings.Fields(strings.Join(rows, " ")), "%s: the words differ from a single row's", at)
			}
		}
	}
}

// TestStatusBarFitsALongFileNameIntoItsRow renders the bar over a file whose
// name is wider than the bar: the name is cut to fill the last row, so the
// read-only badge and the variable count still show beside it.
func TestStatusBarFitsALongFileNameIntoItsRow(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})
	name := ".env." + strings.Repeat("x", 200)
	for _, width := range []int{MinTerminalWidth, 80, 120} {
		sb := NewStatusBarModel()
		sb.Width = width
		sb.ReadOnly = true
		rows := strings.Split(stripAnsi(sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, name, 16)), "\n")
		last := strings.TrimRight(rows[len(rows)-1], " ")
		assert.Regexp(t, `^   \[READ-ONLY\]  \.env\.x+\.\.  16 vars$`, last, "width %d", width)
		assert.Equal(t, width-1, lipgloss.Width(last), "width %d: the file info is not right-aligned", width)
		for _, row := range rows[:len(rows)-1] {
			assert.NotContains(t, row, ".env.", "width %d: the name starts on an earlier row", width)
		}
	}
}

func TestStatusBarNoReadOnlyBadge(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusFiles, ".env", 5)
	assert.NotContains(t, view, "READ-ONLY")
}

func TestStatusBarMessageOverridesHints(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetMessage("File saved successfully")
	theme := BuildTheme(true, config.ColorConfig{})

	view := sb.View(theme, DefaultKeyMap(), ModeNormal, FocusVars, ".env", 5)
	assert.Contains(t, view, "File saved successfully")
}

// SGR fragments of the primary, warning and error colors in statusColorTheme,
// used to tell which of them a status-bar message was rendered in.
const (
	statusPrimarySGR = "38;2;0;255;0"
	statusWarningSGR = "38;2;0;0;255"
	statusErrorSGR   = "38;2;255;0;0"
)

func statusColorTheme() Theme {
	return BuildTheme(true, config.ColorConfig{Primary: "#00FF00", Warning: "#0000FF", Error: "#FF0000"})
}

// assertStatusColor checks that view renders its message in want, one of the
// status*SGR fragments, and in neither of the other two colors.
func assertStatusColor(t *testing.T, view, want string) {
	t.Helper()
	for _, sgr := range []string{statusPrimarySGR, statusWarningSGR, statusErrorSGR} {
		if sgr == want {
			assert.Contains(t, view, sgr)
		} else {
			assert.NotContains(t, view, sgr)
		}
	}
}

// statusView renders app's status bar with statusColorTheme.
func statusView(app App) string {
	return app.statusBar.View(statusColorTheme(), app.keys, app.mode, app.focus, "", 0)
}

func TestStatusBarWarningRendersInWarningColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetWarning("File already exists: .env")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "File already exists: .env")
	assertStatusColor(t, view, statusWarningSGR)
}

func TestStatusBarAlertRendersInErrorColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetAlert("Error saving: disk full")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "Error saving: disk full")
	assertStatusColor(t, view, statusErrorSGR)
}

func TestStatusBarMessageAfterAlertRendersInPrimaryColor(t *testing.T) {
	sb := NewStatusBarModel()
	sb.Width = 120
	sb.SetAlert("Error saving: disk full")
	sb.SetMessage("Saved .env")

	view := sb.View(statusColorTheme(), DefaultKeyMap(), ModeNormal, FocusVars, "", 0)
	assert.Contains(t, view, "Saved .env")
	assertStatusColor(t, view, statusPrimarySGR)
}

func TestStatusBarHintsNameTheKeysTheKeyMapBinds(t *testing.T) {
	keys := DefaultKeyMap()
	keys.Save = key.NewBinding(key.WithKeys("s"))
	keys.Enter = key.NewBinding(key.WithKeys("ctrl+j"))
	keys.Quit = key.NewBinding(key.WithKeys("x", "ctrl+c"))
	keys.Deny = key.NewBinding(key.WithKeys("z"))
	sb := NewStatusBarModel()
	sb.Width = 200
	theme := BuildTheme(true, config.ColorConfig{})
	hint := func(mode AppMode, focus Focus) string {
		return stripAnsi(sb.View(theme, keys, mode, focus, ".env", 5, DiffStats{}))
	}

	for _, focus := range []Focus{FocusFiles, FocusVars} {
		assert.Contains(t, hint(ModeNormal, focus), "[s]save")
		assert.NotContains(t, hint(ModeNormal, focus), "[w]", "the old key is gone")
	}
	assert.Contains(t, hint(ModeComparing, FocusVars), "[s]save")
	assert.Contains(t, hint(ModeMatrix, FocusVars), "[x]back")
	assert.Contains(t, hint(ModeCompareSelect, FocusFiles), "Ctrl+J to pick, Esc/x to cancel")
	assert.Contains(t, hint(ModeEditing, FocusVars), "Ctrl+J to confirm")
	assert.Contains(t, hint(ModeConfirmDelete, FocusVars), "(y/z)")
}
