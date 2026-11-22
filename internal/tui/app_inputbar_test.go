package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/model"
)

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

const inputTail = "TAIL_12345"

var longInputUnderSearchLimit = strings.Repeat("x", 90) + inputTail

func openInputBar(t *testing.T, tc inputBarCase, width int, value string) App {
	t.Helper()
	app := tc.open(resize(newTestApp(layoutFiles()), width, 24))
	require.Equal(t, tc.mode, app.mode, "%s: the bar should be open", tc.name)
	input := tc.input(&app)
	input.SetValue(value)
	input.CursorEnd()
	return app
}

func TestInputBarsScrollALongValueOnOneRow(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, width := range gridWidths {
				short := openInputBar(t, tc, width, "short").View().Content
				view := openInputBar(t, tc, width, longInputUnderSearchLimit).View().Content

				assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "width %d: a long value must not wrap the bar", width)
				assert.Contains(t, stripAnsi(view), inputTail, "width %d: the end of the value, at the cursor, should be in view", width)
			}
		})
	}
}

func TestInputBarsKeepOneRowWithGraphemesTheInputUndercounts(t *testing.T) {
	const variationSelectorEmoji = "\u2764\uFE0F"
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			short := openInputBar(t, tc, 80, "short").View().Content
			view := openInputBar(t, tc, 80, strings.Repeat(variationSelectorEmoji, 40)).View().Content

			assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "the bar should keep one row")
		})
	}
}

func TestInputBarsRefitAfterAResize(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			short := openInputBar(t, tc, 60, "short").View().Content
			view := resize(openInputBar(t, tc, 120, longInputUnderSearchLimit), 60, 24).View().Content

			assert.Equal(t, lipgloss.Height(short), lipgloss.Height(view), "the bar should keep one row")
			assert.Contains(t, stripAnsi(view), inputTail, "the end of the value, at the cursor, should be in view")
		})
	}
}

func TestInputBarsKeepTextTypedMidValueInView(t *testing.T) {
	longUnderSearchLimit := strings.Repeat("x", 90)
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			app := openInputBar(t, tc, 80, longUnderSearchLimit)
			tc.input(&app).SetCursor(10)

			app = press(app, "A", "B", "C")

			require.Equal(t, strings.Repeat("x", 10)+"ABC"+strings.Repeat("x", 80), tc.input(&app).Value())
			assert.Contains(t, stripAnsi(app.View().Content), "ABC", "the text just typed should stay in view")
		})
	}
}

func TestInputBarCutsALongLabel(t *testing.T) {
	const longKey = "UPSTREAM_SERVICE_CIRCUIT_BREAKER_THRESHOLD_MS"
	app := press(resize(newTestApp([]*model.EnvFile{
		makeTestFile(".env", "A"),
		makeTestFile(".env.production.local", "A", longKey),
	}), 60, 20), "m", "j", "a")
	require.Equal(t, ModeMatrixEditing, app.mode, "the matrix should add the long key, missing from .env")

	rows := strings.Split(stripAnsi(app.View().Content), "\n")
	bar := rowDrawing(t, app.View().Content, "Add UPSTREAM")

	assert.Contains(t, rows[bar], "..", "a label wider than the bar should be cut")
	assert.Contains(t, rows[bar+1], "Enter to confirm", "the status bar should follow the one-row editor bar")
}

func TestInputBarsShowTheWholePlaceholder(t *testing.T) {
	for _, tc := range inputBarCases {
		t.Run(tc.name, func(t *testing.T) {
			app := openInputBar(t, tc, 80, "")
			placeholder := tc.input(&app).Placeholder
			require.NotEmpty(t, placeholder)

			assert.Contains(t, stripAnsi(app.View().Content), placeholder, "the whole placeholder, not its first letter, should show")
		})
	}
}
