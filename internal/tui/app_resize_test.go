package tui

import (
	"fmt"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

// resizeFixture returns 30 files that each hold the keys K00 to K29: more
// rows and columns than any view shows at the sizes the resize tests use.
func resizeFixture() []*model.EnvFile {
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	files := make([]*model.EnvFile, 30)
	for i := range files {
		files[i] = makeTestFile(fmt.Sprintf(".env.%02d", i), keys...)
	}
	return files
}

// TestResizeKeepsTheCursorRowInView moves the cursor of every scrollable view
// to a row near the bottom (a column near the right, in the matrix) of a large
// terminal, then shrinks the terminal: the cursor stays in view.
func TestResizeKeepsTheCursorRowInView(t *testing.T) {
	files := resizeFixture()
	down := slices.Repeat([]string{"j"}, 20)
	right := slices.Repeat([]string{"l"}, 20)
	tests := []struct {
		name, atCursor string
		keys           []string
	}{
		{"var list", "K20", append([]string{"right"}, down...)},
		{"file list", "● .env.20", down},
		{"compare", "K20", append([]string{"c", "j", "enter"}, down...)},
		{"matrix row", "K20", append([]string{"m"}, down...)},
		{"matrix column", ".env.20", append([]string{"m"}, right...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := newTestApp(files).Update(tea.WindowSizeMsg{Width: 120, Height: 30})
			app := press(updated.(App), tt.keys...)
			assert.Contains(t, stripAnsi(app.View().Content), tt.atCursor, "the cursor should be in view before the resize")

			updated, _ = app.Update(tea.WindowSizeMsg{Width: 60, Height: 12})

			assert.Contains(t, stripAnsi(updated.(App).View().Content), tt.atCursor, "the cursor should stay in view")
		})
	}
}

// TestGrowingTheTerminalScrollsBackToFillTheView scrolls every scrollable view
// to its last row in a small terminal, then grows the terminal: the view
// scrolls back to show the rows above instead of leaving blank rows below.
func TestGrowingTheTerminalScrollsBackToFillTheView(t *testing.T) {
	files := resizeFixture()
	end := slices.Repeat([]string{"j"}, 29)
	tests := []struct {
		name, firstRow string
		keys           []string
	}{
		{"var list", "K00", append([]string{"right"}, end...)},
		{"file list", ".env.00", end},
		{"compare", "K00", append([]string{"c", "j", "enter"}, end...)},
		{"matrix", "K00", append([]string{"m"}, end...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated, _ := newTestApp(files).Update(tea.WindowSizeMsg{Width: 120, Height: 12})
			app := press(updated.(App), tt.keys...)
			assert.NotContains(t, stripAnsi(app.View().Content), tt.firstRow, "the first row should be scrolled off before the resize")

			updated, _ = app.Update(tea.WindowSizeMsg{Width: 120, Height: 60})

			assert.Contains(t, stripAnsi(updated.(App).View().Content), tt.firstRow, "a view tall enough for every row should show the first")
		})
	}
}

// TestStartupFocusesTheFileList lays out a fresh App: the file list, which
// holds the focus, is drawn focused.
func TestStartupFocusesTheFileList(t *testing.T) {
	updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env", "A")}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := updated.(App)

	assert.Equal(t, FocusFiles, app.focus)
	assert.True(t, app.fileList.Focused, "the file list should be drawn focused")
	assert.False(t, app.varList.Focused)
}

// TestMatrixRendersInATinyTerminal opens the matrix in a one-column terminal,
// narrower than its padding: it renders without panicking. The App draws the
// too-small notice there instead, so the matrix is rendered directly.
func TestMatrixRendersInATinyTerminal(t *testing.T) {
	updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B")}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := press(updated.(App), "m")
	updated, _ = app.Update(tea.WindowSizeMsg{Width: 1, Height: 24})
	app = updated.(App)

	assert.NotPanics(t, func() { _ = app.matrixView.View(app.theme) })
}

// TestTooSmallTerminalShowsANotice shrinks the terminal below the minimum, in
// width or in height: the App draws a notice that fits it instead of its
// panels, and draws the panels again once the terminal is back to the minimum.
func TestTooSmallTerminalShowsANotice(t *testing.T) {
	sizes := []struct{ width, height int }{
		{MinTerminalWidth - 1, 24},
		{80, MinTerminalHeight - 1},
		{30, 5},
	}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env", "A")}).Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			view := updated.(App).View().Content

			assert.Contains(t, stripAnsi(view), "too small")
			assert.NotContains(t, stripAnsi(view), ".env", "the panels should not be drawn")
			assert.LessOrEqual(t, lipgloss.Width(view), size.width, "the notice should fit the terminal")
			assert.LessOrEqual(t, lipgloss.Height(view), size.height, "the notice should fit the terminal")

			updated, _ = updated.Update(tea.WindowSizeMsg{Width: MinTerminalWidth, Height: MinTerminalHeight})

			assert.Contains(t, stripAnsi(updated.(App).View().Content), ".env", "the panels should be back")
		})
	}
}

// TestTooSmallTerminalIgnoresInputButCtrlC presses keys and clicks while the
// terminal is below the minimum: none acts on the hidden screen, and Ctrl+C
// still quits.
func TestTooSmallTerminalIgnoresInputButCtrlC(t *testing.T) {
	updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env", "A", "B")}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := onVars(updated.(App))
	updated, _ = app.Update(tea.WindowSizeMsg{Width: MinTerminalWidth - 1, Height: 24})
	app = updated.(App)

	app = press(app, "j", "d")
	updated, cmd := app.Update(tea.MouseClickMsg{Button: tea.MouseLeft, X: 2, Y: 3})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Equal(t, ModeNormal, app.mode, "d should not open the delete confirmation")
	assert.Equal(t, FocusVars, app.focus, "the click should not move the focus to the file list")
	assert.Equal(t, 0, app.varList.Cursor, "j should not move the cursor")

	_, cmd = app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd())
}
