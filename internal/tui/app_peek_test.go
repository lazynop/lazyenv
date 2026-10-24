package tui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
)

func TestPeekToggle(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars

	assert.False(t, app.varList.Peeking)

	updated, _ := app.Update(keyMsg("p"))
	app = updated.(App)
	assert.True(t, app.varList.Peeking, "first press should enable peek")

	updated, _ = app.Update(keyMsg("p"))
	app = updated.(App)
	assert.False(t, app.varList.Peeking, "second press should disable peek")
}

func TestPeekNoOpOnFilePanel(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusFiles

	updated, _ := app.Update(keyMsg("p"))
	app = updated.(App)
	assert.False(t, app.varList.Peeking, "peek should not activate on file panel")
}

func TestPeekShowsOriginalForModifiedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Edit the variable.
	f.UpdateVar(0, "new_value", false)
	assert.Equal(t, "val_FOO", f.Vars[0].OriginalValue)

	// Enable peek.
	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show original value")
}

func TestPeekShowsNewVariableHint(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Add a new variable and move cursor to it.
	f.AddVar("NEW_VAR", "some_val", false)
	app.varList.Refresh()
	app.varList.MoveDown() // cursor on NEW_VAR

	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.Contains(t, view, "new variable", "peek should show 'new variable' for added vars")
}

func TestPeekShowsNothingForUnmodifiedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.focus = FocusVars
	app.varList.Focused = true
	app.varList.Width = 80
	app.varList.Height = 20

	// Enable peek without editing anything.
	app.varList.Peeking = true
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.NotContains(t, view, "was:", "peek should not show anything for unmodified vars")
	assert.NotContains(t, view, "new variable")
}

func TestPeekHiddenWhenNotFocused(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp(nil)
	app.fileList.Files = append(app.fileList.Files, f)
	app.varList.SetFile(f)
	app.varList.Width = 80
	app.varList.Height = 20

	f.UpdateVar(0, "new_value", false)

	app.varList.Peeking = true
	app.varList.Focused = false // not focused
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.NotContains(t, view, "was:", "peek line should only show when panel is focused")
}

func TestDeletedVarsShownInVarList(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(1) // delete BAR
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "BAR", "deleted var should still appear in the list")
	assert.Contains(t, view, "-", "deleted var should show - marker")
}

func TestDeletedVarsDisappearAfterReAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(1) // delete BAR
	vlm.Refresh()
	f.AddVar("BAR", "new_val", false) // re-add BAR
	vlm.Refresh()

	assert.Empty(t, f.DeletedVars, "re-added var should be removed from DeletedVars")
}

func TestPeekAfterDeleteAndReAdd(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	var vlm VarListModel
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(0)                      // delete FOO (original value: val_FOO)
	f.AddVar("FOO", "new_value", false) // re-add with different value
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show original value after delete+re-add")
}

func TestPeekMasksOriginalOfMaskedSecret(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.UpdateVar(0, "hunter3", true)

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: ••••••••", "peek should mask a masked secret's original value")
	assert.NotContains(t, view, "hunter2", "peek must not reveal a masked secret's original value")
}

func TestPeekShowsOriginalOfRevealedSecret(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.SetShowSecrets(true)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.UpdateVar(0, "hunter3", true)

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: hunter2", "peek should show a revealed secret's original value")
}

func TestPeekFlattensOriginalControlChars(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	f.Vars[0].Value = "line1\nline2\tend"
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.UpdateVar(0, "new_value", false)

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: line1↵line2⇥end", "peek should flatten the original value to one row")
}

func TestPeekMasksOriginalOfSecretReAddedWithPlainValue(t *testing.T) {
	f := makeTestFile(".env", "GREETING")
	f.Vars[0].Value, f.Vars[0].IsSecret = "ghp_abc123", true // secret by value only
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(0)
	f.AddVar("GREETING", "hello", false)
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: ••••••••", "peek should mask the original value of a re-added secret")
	assert.NotContains(t, view, "ghp_abc123", "peek must not reveal the original value of a re-added secret")
	assert.Contains(t, view, "hello", "the re-added plain value should stay in clear")
}

func TestPeekMasksOriginalOfSecretRenamedToPlainKey(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true // secret by key only
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	f.UpdateVar(0, "x", true)

	app.mode = ModeEditing
	app.editor.StartEditKey(&f.Vars[0], 0)
	app.editor.input.SetValue("GREETING")
	updated, _ := app.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	app = updated.(App)
	require.False(t, f.Vars[0].IsSecret, "the renamed var is no longer a secret")

	app.varList.Focused = true
	app.varList.Peeking = true
	app.varList.Width = 80
	app.varList.Height = 20
	theme := BuildTheme(true, config.ColorConfig{})
	view := app.varList.View(theme)

	assert.Contains(t, view, "was: ••••••••", "peek should mask an original value that was a secret")
	assert.NotContains(t, view, "hunter2", "peek must not reveal an original value that was a secret")
}

func TestPeekShowsUnchangedValueOfRenamedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.RenameVar(0, "BAR", false)
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show the value a renamed var kept")
}

func TestPeekShowsOriginalOfRenamedThenEditedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.RenameVar(0, "BAR", false)
	f.UpdateVar(0, "new_value", false)
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show the value from before the rename and the edit")
}

func TestPeekShowsOnDiskOriginalOfEditedThenReAddedVar(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.UpdateVar(0, "edited", false)
	f.DeleteVar(0)
	f.AddVar("FOO", "new_value", false)
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show the value on disk, not the edit made before the delete")
}

// peekBottomRowList returns a var list three rows tall over five vars, so the
// last var only fits on the bottom row.
func peekBottomRowList(f *model.EnvFile) VarListModel {
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = panelChromeHeight + 3
	return vlm
}

func TestPeekKeepsCursorRowVisibleAtTheBottom(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})

	t.Run("moving onto a new var", func(t *testing.T) {
		f := makeTestFile(".env", "A", "B", "C", "D")
		f.AddVar("NEW_VAR", "value", false)
		vlm := peekBottomRowList(f)
		vlm.Peeking = true
		for range 4 {
			vlm.MoveDown()
		}
		view := vlm.View(theme)
		assert.Contains(t, view, "NEW_VAR")
		assert.Contains(t, view, "new variable")
	})

	t.Run("turning peek on", func(t *testing.T) {
		f := makeTestFile(".env", "A", "B", "C", "D", "E")
		f.UpdateVar(2, "edited", false)
		vlm := peekBottomRowList(f)
		vlm.SetCursor(2) // bottom row
		vlm.TogglePeek()
		view := vlm.View(theme)
		assert.Contains(t, view, "edited")
		assert.Contains(t, view, "was: val_C")
	})

	t.Run("editing the var under the cursor", func(t *testing.T) {
		f := makeTestFile(".env", "A", "B", "C", "D", "E")
		vlm := peekBottomRowList(f)
		vlm.Peeking = true
		vlm.SetCursor(2) // bottom row
		f.UpdateVar(2, "edited", false)
		vlm.Refresh()
		view := vlm.View(theme)
		assert.Contains(t, view, "edited")
		assert.Contains(t, view, "was: val_C")
	})
}

func TestPeekReservesNoRowWhenUnfocused(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E")
	f.UpdateVar(0, "edited", false)
	vlm := peekBottomRowList(f)
	vlm.Peeking = true
	vlm.Focused = false

	rows := strings.Split(stripAnsi(vlm.View(BuildTheme(true, config.ColorConfig{}))), "\n")

	assert.Contains(t, rows[len(rows)-2], "val_C", "with no peek line drawn, the last inner row should hold a var")
}

// fourRowPanelsHeight is the terminal height that leaves the panels four
// rows inside their chrome.
const fourRowPanelsHeight = bottomChromeHeight + panelChromeHeight + 4

// peekedBottomRowApp returns an App whose file list has the focus, with peek
// on and the var cursor scrolled, by mouse wheel, onto the bottom row: F,
// edited, below C, D and E.
func peekedBottomRowApp() App {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F")
	f.UpdateVar(5, "edited", false)
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "p", "left")
	for range len(f.Vars) - 1 {
		updated, _ := app.Update(tea.MouseWheelMsg{X: 60, Y: 3, Button: tea.MouseWheelDown})
		app = updated.(App)
	}
	return app
}

// TestFocusingTheVarListKeepsThePeekedCursorRowVisible scrolls the var cursor
// onto the bottom row while the file list has the focus, so no peek line is
// counted, then focuses the var list: the peek line now drawn under the
// cursor row must not push that row out of the panel.
func TestFocusingTheVarListKeepsThePeekedCursorRowVisible(t *testing.T) {
	focusVars := map[string]tea.Msg{
		"right key": tea.KeyPressMsg{Code: tea.KeyRight},
		// The bottom border selects no row, so only the focus changes.
		"click on the bottom border": tea.MouseClickMsg{X: 60, Y: fourRowPanelsHeight - bottomChromeHeight - 1, Button: tea.MouseLeft},
	}
	for name, focus := range focusVars {
		t.Run(name, func(t *testing.T) {
			updated, _ := peekedBottomRowApp().Update(focus)

			view := stripAnsi(updated.(App).View().Content)
			assert.Contains(t, view, "edited", "the cursor row should stay in the panel")
			assert.Contains(t, view, "was: val_F", "the peek line should stay in the panel")
		})
	}
}

// TestPeekLineLeavesRoomForDeletedVars fills a ten-row panel with five vars,
// a peek line and deleted vars: four deleted vars fit below.
func TestPeekLineLeavesRoomForDeletedVars(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "X1", "X2", "X3", "X4", "X5")
	for range 5 {
		f.DeleteVar(5)
	}
	f.UpdateVar(0, "edited", false)
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = panelChromeHeight + 10

	view := stripAnsi(vlm.View(BuildTheme(true, config.ColorConfig{})))

	drawn := 0
	for _, key := range []string{"X1", "X2", "X3", "X4", "X5"} {
		if strings.Contains(view, key) {
			drawn++
		}
	}
	assert.Equal(t, 4, drawn, "the rows left under five vars and the peek line should hold four deleted vars")
}

// lastVarAboveDeletedApp returns an App whose var list, four rows tall and
// focused, has its cursor on H, the last of eight vars, with GONE1 and GONE2
// deleted below it.
func lastVarAboveDeletedApp() App {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F", "G", "H", "GONE1", "GONE2")
	f.DeleteVar(9)
	f.DeleteVar(8)
	return press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "j", "j", "j", "j", "j", "j", "j")
}

// TestDeletedVarsShowAtTheEndOfALongList moves the cursor to the last var of a
// list longer than its panel: the list scrolls on to show the deleted vars
// below it, keeping the cursor row in view.
func TestDeletedVarsShowAtTheEndOfALongList(t *testing.T) {
	app := lastVarAboveDeletedApp()

	view := stripAnsi(app.View().Content)

	assert.Contains(t, view, "val_H", "the cursor row should stay in view")
	assert.Contains(t, view, "GONE1")
	assert.Contains(t, view, "GONE2")
}

// TestMovingUpFromTheLastVarKeepsTheScroll shows the deleted vars from the last
// var, then moves up one var: the list keeps its scroll instead of jumping
// back, since the cursor row is still in view.
func TestMovingUpFromTheLastVarKeepsTheScroll(t *testing.T) {
	app := lastVarAboveDeletedApp()
	offset := app.varList.Offset

	app = press(app, "k") // G

	assert.Equal(t, offset, app.varList.Offset, "the list should not scroll while G is in view")
	assert.Contains(t, stripAnsi(app.View().Content), "GONE2")
}

// TestMoreDeletedVarsThanRowsKeepTheCursorInView deletes more vars than the
// panel has rows, then moves to the last var: the cursor row stays in view,
// with as many deleted vars below it as fit.
func TestMoreDeletedVarsThanRowsKeepTheCursorInView(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "GONE1", "GONE2", "GONE3", "GONE4", "GONE5", "GONE6")
	for range 6 {
		f.DeleteVar(2)
	}
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "j") // B, the last var

	view := stripAnsi(app.View().Content)

	assert.Contains(t, view, "val_B", "the cursor row should stay in view")
	for _, key := range []string{"GONE1", "GONE2", "GONE3"} {
		assert.Contains(t, view, key, "the rows under the cursor should hold deleted vars")
	}
}

// TestDeletingTheLastVarListsItUnderTheNote deletes a file's only var: the
// panel says it has no variables and still lists the deleted one below, as it
// does under the vars that are left.
func TestDeletingTheLastVarListsItUnderTheNote(t *testing.T) {
	app := press(onVars(resize(newTestApp([]*model.EnvFile{makeTestFile(".env", "ONLY")}), 80, 24)), "d", "y")
	require.Equal(t, ModeNormal, app.mode)

	rows := strings.Split(stripAnsi(app.View().Content), "\n")
	note := slices.IndexFunc(rows, func(row string) bool { return strings.Contains(row, "No variables") })
	require.GreaterOrEqual(t, note, 0, "the panel should say it has no variables")
	assert.Contains(t, rows[note+1], "ONLY  val_ONLY", "the deleted var should be listed below, its key whole")
}

// TestALongNoMatchesNoteKeepsItsRow searches, for text no var matches, a file
// whose vars are deleted but one, the query wider than the panel: the query is
// cut so the note keeps to its one row and its closing quote, and the deleted
// vars listed below keep the panel inside the terminal.
func TestALongNoMatchesNoteKeepsItsRow(t *testing.T) {
	keys := []string{"KEEP"}
	for i := range 11 {
		keys = append(keys, fmt.Sprintf("GONE%02d", i))
	}
	f := makeTestFile(".env", keys...)
	for range 11 {
		f.DeleteVar(1)
	}
	app := press(onVars(resize(newTestApp([]*model.EnvFile{f}), MinTerminalWidth, 16)), "/")
	app = press(app, slices.Repeat([]string{"z"}, 60)...)

	view := app.View().Content
	assert.LessOrEqual(t, lipgloss.Height(view), 16, "the screen should fit the terminal")
	rows := strings.Split(stripAnsi(view), "\n")
	note := slices.IndexFunc(rows, func(row string) bool { return strings.Contains(row, "No matches for") })
	require.GreaterOrEqual(t, note, 0, "the panel should say nothing matches")
	assert.True(t, strings.HasSuffix(strings.TrimRight(rows[note], "│ "), `.."`), "the cut query should keep its closing quote: %s", rows[note])
	assert.Contains(t, rows[note+1], "GONE00", "the deleted vars should follow the note's one row")
}
