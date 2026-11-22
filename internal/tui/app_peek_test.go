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

	f.UpdateVar(0, "new_value")
	assert.Equal(t, "val_FOO", f.Vars[0].OriginalValue)

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

	f.AddVar("NEW_VAR", "some_val")
	app.varList.Refresh()
	app.varList.MoveDown()

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

	f.UpdateVar(0, "new_value")

	app.varList.Peeking = true
	app.varList.Focused = false
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

	f.DeleteVar(1)
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

	f.DeleteVar(1)
	vlm.Refresh()
	f.AddVar("BAR", "new_val")
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

	f.DeleteVar(0)
	f.AddVar("FOO", "new_value")
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

	f.UpdateVar(0, "hunter3")

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

	f.UpdateVar(0, "hunter3")

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

	f.UpdateVar(0, "new_value")

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: line1↵line2⇥end", "peek should flatten the original value to one row")
}

func TestPeekMasksOriginalOfSecretReAddedWithPlainValue(t *testing.T) {
	const secretByValueOnly = "ghp_abc123"
	f := makeTestFile(".env", "GREETING")
	f.Vars[0].Value, f.Vars[0].IsSecret = secretByValueOnly, true
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Peeking = true
	vlm.Width = 80
	vlm.Height = 20

	f.DeleteVar(0)
	f.AddVar("GREETING", "hello")
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: ••••••••", "peek should mask the original value of a re-added secret")
	assert.NotContains(t, view, secretByValueOnly, "peek must not reveal the original value of a re-added secret")
	assert.Contains(t, view, "hello", "the re-added plain value should stay in clear")
}

func TestPeekMasksOriginalOfSecretRenamedToPlainKey(t *testing.T) {
	const secretByKeyOnly = "API_TOKEN"
	f := makeTestFile(".env", secretByKeyOnly)
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	f.UpdateVar(0, "x")

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

	f.RenameVar(0, "BAR")
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

	f.RenameVar(0, "BAR")
	f.UpdateVar(0, "new_value")
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

	f.UpdateVar(0, "edited")
	f.DeleteVar(0)
	f.AddVar("FOO", "new_value")
	vlm.Refresh()

	theme := BuildTheme(true, config.ColorConfig{})
	view := vlm.View(theme)

	assert.Contains(t, view, "was: val_FOO", "peek should show the value on disk, not the edit made before the delete")
}

const (
	peekListRows      = 3
	peekListBottomRow = peekListRows - 1
)

func peekBottomRowList(f *model.EnvFile) VarListModel {
	vlm := NewVarListModel(config.DefaultConfig().Layout)
	vlm.SetFile(f)
	vlm.Focused = true
	vlm.Width = 80
	vlm.Height = panelChromeHeight + peekListRows
	return vlm
}

func TestPeekKeepsCursorRowVisibleAtTheBottom(t *testing.T) {
	theme := BuildTheme(true, config.ColorConfig{})

	t.Run("moving onto a new var", func(t *testing.T) {
		f := makeTestFile(".env", "A", "B", "C", "D")
		f.AddVar("NEW_VAR", "value")
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
		f.UpdateVar(peekListBottomRow, "edited")
		vlm := peekBottomRowList(f)
		vlm.SetCursor(peekListBottomRow)
		vlm.TogglePeek()
		view := vlm.View(theme)
		assert.Contains(t, view, "edited")
		assert.Contains(t, view, "was: val_C")
	})

	t.Run("editing the var under the cursor", func(t *testing.T) {
		f := makeTestFile(".env", "A", "B", "C", "D", "E")
		vlm := peekBottomRowList(f)
		vlm.Peeking = true
		vlm.SetCursor(peekListBottomRow)
		f.UpdateVar(peekListBottomRow, "edited")
		vlm.Refresh()
		view := vlm.View(theme)
		assert.Contains(t, view, "edited")
		assert.Contains(t, view, "was: val_C")
	})
}

func TestPeekReservesNoRowWhenUnfocused(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E")
	f.UpdateVar(0, "edited")
	vlm := peekBottomRowList(f)
	vlm.Peeking = true
	vlm.Focused = false

	rows := strings.Split(stripAnsi(vlm.View(BuildTheme(true, config.ColorConfig{}))), "\n")

	assert.Contains(t, rows[len(rows)-2], "val_C", "with no peek line drawn, the last inner row should hold a var")
}

const fourRowPanelsHeight = bottomChromeHeight + panelChromeHeight + 4

func peekedBottomRowApp() App {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F")
	f.UpdateVar(5, "edited")
	peekThenFocusTheFileList := []string{"right", "p", "left"}
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), peekThenFocusTheFileList...)
	for range len(f.Vars) - 1 {
		updated, _ := app.Update(tea.MouseWheelMsg{X: 60, Y: 3, Button: tea.MouseWheelDown})
		app = updated.(App)
	}
	return app
}

func TestFocusingTheVarListKeepsThePeekedCursorRowVisible(t *testing.T) {
	focusVars := map[string]tea.Msg{
		"right key": tea.KeyPressMsg{Code: tea.KeyRight},
		"click on the bottom border, which selects no row": tea.MouseClickMsg{X: 60, Y: fourRowPanelsHeight - bottomChromeHeight - 1, Button: tea.MouseLeft},
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

func TestPeekLineLeavesRoomForDeletedVars(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "X1", "X2", "X3", "X4", "X5")
	for range 5 {
		f.DeleteVar(5)
	}
	f.UpdateVar(0, "edited")
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

func lastVarAboveDeletedApp() App {
	f := makeTestFile(".env", "A", "B", "C", "D", "E", "F", "G", "H", "GONE1", "GONE2")
	f.DeleteVar(9)
	f.DeleteVar(8)
	return press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "j", "j", "j", "j", "j", "j", "j")
}

func TestDeletedVarsShowAtTheEndOfALongList(t *testing.T) {
	app := lastVarAboveDeletedApp()

	view := stripAnsi(app.View().Content)

	assert.Contains(t, view, "val_H", "the cursor row should stay in view")
	assert.Contains(t, view, "GONE1")
	assert.Contains(t, view, "GONE2")
}

func TestMovingUpFromTheLastVarKeepsTheScroll(t *testing.T) {
	app := lastVarAboveDeletedApp()
	offset := app.varList.Offset

	app = press(app, "k")

	assert.Equal(t, offset, app.varList.Offset, "the list should not scroll while G is in view")
	assert.Contains(t, stripAnsi(app.View().Content), "GONE2")
}

func TestMoreDeletedVarsThanRowsKeepTheCursorInView(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "GONE1", "GONE2", "GONE3", "GONE4", "GONE5", "GONE6")
	for range 6 {
		f.DeleteVar(2)
	}
	app := press(resize(newTestApp([]*model.EnvFile{f}), 80, fourRowPanelsHeight), "right", "j")

	view := stripAnsi(app.View().Content)

	assert.Contains(t, view, "val_B", "the cursor row should stay in view")
	for _, key := range []string{"GONE1", "GONE2", "GONE3"} {
		assert.Contains(t, view, key, "the rows under the cursor should hold deleted vars")
	}
}

func TestDeletingTheLastVarListsItUnderTheNote(t *testing.T) {
	app := press(onVars(resize(newTestApp([]*model.EnvFile{makeTestFile(".env", "ONLY")}), 80, 24)), "d", "y")
	require.Equal(t, ModeNormal, app.mode)

	rows := strings.Split(stripAnsi(app.View().Content), "\n")
	note := slices.IndexFunc(rows, func(row string) bool { return strings.Contains(row, "No variables") })
	require.GreaterOrEqual(t, note, 0, "the panel should say it has no variables")
	assert.Contains(t, rows[note+1], "ONLY  val_ONLY", "the deleted var should be listed below, its key whole")
}

func TestALongNoMatchesNoteKeepsItsRow(t *testing.T) {
	keys := []string{"KEEP"}
	for i := range 11 {
		keys = append(keys, fmt.Sprintf("GONE%02d", i))
	}
	f := makeTestFile(".env", keys...)
	for range 11 {
		f.DeleteVar(1)
	}
	const queryWiderThanThePanel = 60
	app := press(onVars(resize(newTestApp([]*model.EnvFile{f}), MinTerminalWidth, 16)), "/")
	app = press(app, slices.Repeat([]string{"z"}, queryWiderThanThePanel)...)

	view := app.View().Content
	assert.LessOrEqual(t, lipgloss.Height(view), 16, "the screen should fit the terminal")
	rows := strings.Split(stripAnsi(view), "\n")
	note := slices.IndexFunc(rows, func(row string) bool { return strings.Contains(row, "No matches for") })
	require.GreaterOrEqual(t, note, 0, "the panel should say nothing matches")
	assert.True(t, strings.HasSuffix(strings.TrimRight(rows[note], "│ "), `.."`), "the cut query should keep its closing quote: %s", rows[note])
	assert.Contains(t, rows[note+1], "GONE00", "the deleted vars should follow the note's one row")
}
