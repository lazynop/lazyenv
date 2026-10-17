package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

// inputBarCase opens one input bar over layoutFiles and returns the text
// input it holds.
type inputBarCase struct {
	name  string
	mode  AppMode
	open  func(app App) App
	input func(app *App) *textinput.Model
}

var inputBarCases = []inputBarCase{
	{"search", ModeSearching, func(app App) App { return press(onVars(app), "/") },
		func(app *App) *textinput.Model { return &app.searchInput }},
	{"edit value", ModeEditing, func(app App) App { return press(onVars(app), "j", "j", "e") },
		func(app *App) *textinput.Model { return &app.editor.input }},
	{"new file", ModeCreateFile, func(app App) App { return press(app, "N") },
		func(app *App) *textinput.Model { return &app.fileInput }},
	{"compare edit", ModeEditingCompare, func(app App) App { return press(app, "c", "j", "enter", "j", "j", "e") },
		func(app *App) *textinput.Model { return &app.editor.input }},
	{"matrix add", ModeMatrixEditing, func(app App) App { return press(app, "m", "j", "right", "a") },
		func(app *App) *textinput.Model { return &app.matrixView.editor }},
}

// inputTail ends longInput, a value wider than any input bar in the layout
// grid and within the search input's 100-rune limit.
const inputTail = "TAIL_12345"

var longInput = strings.Repeat("x", 90) + inputTail

// openInputBar sizes an App over layoutFiles to a terminal width columns wide,
// opens tc's bar and types value into it.
func openInputBar(t *testing.T, tc inputBarCase, width int, value string) App {
	t.Helper()
	updated, _ := newTestApp(layoutFiles()).Update(tea.WindowSizeMsg{Width: width, Height: 24})
	app := tc.open(updated.(App))
	require.Equal(t, tc.mode, app.mode, "%s: the bar should be open", tc.name)
	input := tc.input(&app)
	input.SetValue(value)
	input.CursorEnd()
	return app
}

// TestInputBarsScrollALongValueOnOneRow types a value wider than the
// terminal into every input bar, at every width of the layout grid: the bar
// keeps one row, scrolling the value so its end, where the cursor is, stays in
// view.
func TestInputBarsScrollALongValueOnOneRow(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, width := range gridWidths {
				short := openInputBar(t, tc, width, "short").View().Content
				view := openInputBar(t, tc, width, longInput).View().Content

				assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "width %d: a long value must not wrap the bar", width)
				assert.Contains(t, stripAnsi(view), inputTail, "width %d: the end of the value, at the cursor, should be in view", width)
			}
		})
	}
}

// TestInputBarsKeepOneRowWithWideGraphemes types emoji with a variation
// selector, which the text input measures as one column and the terminal draws
// in two: the bar still keeps one row.
func TestInputBarsKeepOneRowWithWideGraphemes(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			short := openInputBar(t, tc, 80, "short").View().Content
			view := openInputBar(t, tc, 80, strings.Repeat("\u2764\uFE0F", 40)).View().Content

			assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "the bar should keep one row")
		})
	}
}

// TestInputBarsRefitAfterAResize shrinks the terminal while every input bar
// holds a long value: the bar refits to the new width and keeps one row.
func TestInputBarsRefitAfterAResize(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			short := openInputBar(t, tc, 60, "short").View().Content
			updated, _ := openInputBar(t, tc, 120, longInput).Update(tea.WindowSizeMsg{Width: 60, Height: 24})

			view := updated.(App).View().Content

			assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "the bar should keep one row")
			assert.Contains(t, stripAnsi(view), inputTail, "the end of the value, at the cursor, should be in view")
		})
	}
}

// TestInputBarsKeepTextTypedMidValueInView moves the cursor near the start of
// a value wider than the bar and types there: the text typed stays in view.
func TestInputBarsKeepTextTypedMidValueInView(t *testing.T) {
	long := strings.Repeat("x", 90) // room to type within the search input's 100-rune limit
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			app := openInputBar(t, tc, 80, long)
			tc.input(&app).SetCursor(10)

			app = press(app, "A", "B", "C")

			require.Equal(t, strings.Repeat("x", 10)+"ABC"+strings.Repeat("x", 80), tc.input(&app).Value())
			assert.Contains(t, stripAnsi(app.View().Content), "ABC", "the text just typed should stay in view")
		})
	}
}

// TestInputBarCutsALongLabel opens the matrix editor bar on a key whose label
// is wider than the bar: the label is cut, so the bar keeps one row.
func TestInputBarCutsALongLabel(t *testing.T) {
	const longKey = "UPSTREAM_SERVICE_CIRCUIT_BREAKER_THRESHOLD_MS"
	updated, _ := newTestApp([]*model.EnvFile{
		makeTestFile(".env", "A"),
		makeTestFile(".env.production.local", "A", longKey),
	}).Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	app := press(updated.(App), "m", "j", "a") // the long key, missing from .env
	require.Equal(t, ModeMatrixEditing, app.mode)

	rows := strings.Split(stripAnsi(app.View().Content), "\n")
	bar := rowDrawing(t, app.View().Content, "Add UPSTREAM")

	assert.Contains(t, rows[bar], "..", "a label wider than the bar should be cut")
	assert.Contains(t, rows[bar+1], "Enter to confirm", "the status bar should follow the one-row editor bar")
}

// TestInputBarsShowTheWholePlaceholder opens every input bar empty: it shows
// its whole placeholder, not the first letter.
func TestInputBarsShowTheWholePlaceholder(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			app := openInputBar(t, tc, 80, "")
			placeholder := tc.input(&app).Placeholder
			require.NotEmpty(t, placeholder)

			assert.Contains(t, stripAnsi(app.View().Content), placeholder)
		})
	}
}
