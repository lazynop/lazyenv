package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/lazynop/lazyenv/internal/model"
)

// TestResizeKeepsTheCursorRowInView moves the cursor of every scrollable view
// to a row near the bottom (a column near the right, in the matrix) of a large
// terminal, then shrinks the terminal: the cursor stays in view.
func TestResizeKeepsTheCursorRowInView(t *testing.T) {
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	files := make([]*model.EnvFile, 30)
	for i := range files {
		files[i] = makeTestFile(fmt.Sprintf(".env.%02d", i), keys...)
	}
	down := strings.Split(strings.Repeat("j", 20), "")
	right := strings.Split(strings.Repeat("l", 20), "")
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
	keys := make([]string, 30)
	for i := range keys {
		keys[i] = fmt.Sprintf("K%02d", i)
	}
	files := make([]*model.EnvFile, 30)
	for i := range files {
		files[i] = makeTestFile(fmt.Sprintf(".env.%02d", i), keys...)
	}
	end := strings.Split(strings.Repeat("j", 29), "")
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

// TestMatrixRendersInATinyTerminal opens the matrix in a one-column terminal:
// it renders without panicking.
func TestMatrixRendersInATinyTerminal(t *testing.T) {
	updated, _ := newTestApp([]*model.EnvFile{makeTestFile(".env", "A"), makeTestFile(".env.prod", "B")}).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app := press(updated.(App), "m")
	updated, _ = app.Update(tea.WindowSizeMsg{Width: 1, Height: 24})

	assert.NotPanics(t, func() { _ = updated.(App).View() })
}
