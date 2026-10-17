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
