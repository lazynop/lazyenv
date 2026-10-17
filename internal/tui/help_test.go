package tui

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

// openHelp sizes a new app to the terminal and opens the help screen.
func openHelp(width, height int) App {
	return openHelpWith(config.DefaultConfig(), width, height)
}

// openHelpWith is openHelp for an app started with cfg.
func openHelpWith(cfg config.Config, width, height int) App {
	var app tea.Model = NewApp(cfg, StartupMessages{})
	app, _ = app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return press(app.(App), "?")
}

// scrollPastTheEnd is more down presses than the help has rows to scroll at
// any size the layout tests render, the 60x10 minimum included.
const scrollPastTheEnd = 120

// helpLastRow is the description on the last row of the help.
const helpLastRow = "Jump to the top/bottom"

// helpScreen is the help screen as plain text.
func helpScreen(app App) string {
	return stripAnsi(app.View().Content)
}

func TestViewHelp(t *testing.T) {
	help := helpScreen(openHelp(100, 40))
	assert.Contains(t, help, "lazyenv")
	assert.Contains(t, help, "Files panel")
	assert.Contains(t, help, "Variables panel")
	assert.Contains(t, help, "Esc ? q to close")
}

func TestHelpScrollsDownToItsLastRow(t *testing.T) {
	app := openHelp(80, 24)
	assert.Contains(t, helpScreen(app), "lazyenv — TUI for managing .env files")
	assert.NotContains(t, helpScreen(app), helpLastRow)

	for range scrollPastTheEnd {
		app = press(app, "down")
	}

	screen := helpScreen(app)
	assert.Equal(t, 24, lipgloss.Height(screen))
	assert.Contains(t, screen, helpLastRow, "the last row is reachable")
	assert.NotContains(t, screen, "lazyenv — TUI for managing .env files")
	assert.Contains(t, screen, "Esc ? q to close", "the footer stays on screen")
}

func TestHelpScrollStopsAtBothEnds(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)
	assert.Equal(t, top, helpScreen(press(app, "up")), "up at the top does nothing")

	for range scrollPastTheEnd {
		app = press(app, "j")
	}
	bottom := helpScreen(app)
	assert.Equal(t, bottom, helpScreen(press(app, "j")), "down at the bottom does nothing")
	assert.NotEqual(t, bottom, helpScreen(press(app, "k")), "up from the bottom scrolls back")
}

func TestHelpReopensAtTheTop(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)

	app = press(app, "down", "down", "down", "esc", "?")

	assert.Equal(t, top, helpScreen(app))
}

func TestHelpScrollsWithTheMouseWheel(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)

	updated, _ := app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelDown})
	app = updated.(App)
	assert.NotEqual(t, top, helpScreen(app), "the wheel scrolls down")

	updated, _ = app.Update(tea.MouseWheelMsg{X: 1, Y: 5, Button: tea.MouseWheelUp})
	app = updated.(App)
	assert.Equal(t, top, helpScreen(app), "and back up")
}

func TestHelpHintsAtScrollingOnlyWhenItOverflows(t *testing.T) {
	assert.Contains(t, helpScreen(openHelp(80, 24)), "↑/↓ j/k PgUp/PgDn to scroll")

	tall := helpScreen(openHelp(120, 200))
	assert.NotContains(t, tall, "to scroll")
	assert.Contains(t, tall, "lazyenv — TUI", "a terminal tall enough shows the whole help")
	assert.Contains(t, tall, helpLastRow, "a terminal tall enough shows the whole help")
}

func TestHelpGrowingTheTerminalScrollsBackToFit(t *testing.T) {
	app := openHelp(80, 24)
	for range scrollPastTheEnd {
		app = press(app, "down")
	}

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 200})

	assert.Contains(t, helpScreen(updated.(App)), "lazyenv — TUI for managing .env files",
		"a terminal tall enough for the whole help shows it from the top")
}

func TestHelpIgnoresClicks(t *testing.T) {
	app := press(openHelp(80, 24), "down", "down")
	before := helpScreen(app)

	updated, _ := app.Update(tea.MouseClickMsg{X: 10, Y: 5, Button: tea.MouseLeft})
	app = updated.(App)

	assert.Equal(t, ModeHelp, app.mode, "a click keeps the help open")
	assert.Equal(t, before, helpScreen(app), "and where it was scrolled to")
}

func TestHelpFooterNamesEveryCloseAndScrollKeyOnOneRowAtTheMinimumWidth(t *testing.T) {
	overflowing := helpScreen(openHelp(MinTerminalWidth, MinTerminalHeight))
	assert.Contains(t, overflowing, "Press ↑/↓ j/k PgUp/PgDn to scroll, Esc ? q to close")

	fitting := helpScreen(openHelp(MinTerminalWidth, 200))
	assert.Contains(t, fitting, helpLastRow, "a terminal tall enough shows the whole help")
	assert.Contains(t, fitting, "Press Esc ? q to close")
}

// continuationIndent finds the screen row holding description and returns the
// column the description starts at and the column the text of the row below
// starts at.
func continuationIndent(t *testing.T, screen, description string) (descCol, nextCol int) {
	t.Helper()
	rows := strings.Split(screen, "\n")
	for i, row := range rows {
		col := strings.Index(row, description)
		if col < 0 {
			continue
		}
		require.Less(t, i+1, len(rows), "a row follows %q", description)
		next := rows[i+1]
		return lipgloss.Width(row[:col]), lipgloss.Width(next) - lipgloss.Width(strings.TrimLeft(next, " "))
	}
	require.Failf(t, "row not on screen", "%q", description)
	return 0, 0
}

func TestHelpIndentsWrappedRowsUnderTheirDescription(t *testing.T) {
	tall := helpScreen(openHelp(MinTerminalWidth, 200))
	for _, description := range []string{"Create template (keys only", "Reorder variables on disk"} {
		descCol, nextCol := continuationIndent(t, tall, description)
		assert.Equal(t, descCol, nextCol, "the continuation of %q starts under its description", description)
	}

	app := openHelp(MinTerminalWidth, MinTerminalHeight)
	for range scrollPastTheEnd {
		app = press(app, "down")
	}
	screen := helpScreen(app)
	assert.Contains(t, screen, helpLastRow, "scrolling counts the wrapped rows")
}

// helpSections splits a help screen tall enough for the whole text into its
// sections: each title, at the text's left margin, maps to its rows, trimmed,
// each joined with the continuation lines a wrap indents under it.
func helpSections(screen string) map[string][]string {
	keyRow := strings.Repeat(" ", helpPaddingWidth+4)
	sections := map[string][]string{}
	title := ""
	for _, row := range strings.Split(screen, "\n") {
		text := strings.TrimSpace(row)
		rows := sections[title]
		switch {
		case text == "":
		case strings.HasPrefix(row, keyRow+" ") && len(rows) > 0:
			rows[len(rows)-1] += " " + text
		case strings.HasPrefix(row, keyRow):
			sections[title] = append(rows, text)
		default:
			title = text
		}
	}
	return sections
}

// helpRow returns the row of section whose key column is keys, or "".
func helpRow(sections map[string][]string, section, keys string) string {
	for _, row := range sections[section] {
		if column, _, _ := strings.Cut(row, "   "); strings.TrimSpace(column) == keys {
			return row
		}
	}
	return ""
}

func TestHelpGroupsTheKeysByScreen(t *testing.T) {
	sections := helpSections(helpScreen(openHelp(120, 200)))

	assert.Contains(t, helpRow(sections, "Files panel", "N"), "Create new .env file")
	assert.Contains(t, helpRow(sections, "Files panel", "D"), "with confirmation")
	assert.Contains(t, helpRow(sections, "Variables panel", "e"), "Edit variable value")
	assert.Contains(t, helpRow(sections, "Variables panel", "Enter/Space"), "group")
	assert.Contains(t, helpRow(sections, "Both panels", "w"), "Save changes")
	assert.Contains(t, helpRow(sections, "Both panels", "q / Ctrl+C"), "Quit")
	assert.Contains(t, helpRow(sections, "Compare: pick the second file", "Enter"), "Compare")
	assert.Contains(t, helpRow(sections, "Compare: pick the second file", "Esc"), "Cancel")
	assert.Contains(t, helpRow(sections, "Compare: pick the second file", "q / Ctrl+C"), "Cancel")
	assert.Contains(t, helpRow(sections, "Prompts", "Esc"), "clear the filter")
	assert.Contains(t, helpRow(sections, "Variable Indicators", "▸"), "Collapsed group")
	assert.Contains(t, helpRow(sections, "Variable Indicators", "▾"), "Expanded group")
	assert.Contains(t, helpRow(sections, "Compare view", "n/N"), "next/prev difference")
	assert.Contains(t, helpRow(sections, "Compare view", "←/→  h/l"), "Copy value")
	assert.Contains(t, helpRow(sections, "Compare view", "←/→  h/l"), "delete",
		"the copy deletes the key from the target when the source lacks it")
	assert.Contains(t, helpRow(sections, "Prompts", "n"), "yes/no")
	assert.Contains(t, helpRow(sections, "Prompts", "a / g"), "alphabetical / grouped")
	assert.Contains(t, helpRow(sections, "Compare view", "e/E"), "left/right file")
	assert.Contains(t, helpRow(sections, "Compare view", "f"), "differences only")
	assert.Contains(t, helpRow(sections, "Compare view", "q / Ctrl+C"), "Back to main view")
	assert.Contains(t, helpRow(sections, "Matrix", "a"), "Add missing variable")
	assert.Contains(t, helpRow(sections, "Matrix", "o"), "completeness")
	assert.Contains(t, helpRow(sections, "Matrix", "d"), "with confirmation")
	assert.Contains(t, helpRow(sections, "Compare Indicators", "≠"), "Different values")
	assert.Contains(t, helpRow(sections, "Compare Indicators", "+"), "Only in the left file")
	assert.Contains(t, helpRow(sections, "Compare Indicators", "-"), "Only in the right file")
	assert.Contains(t, helpRow(sections, "Matrix Indicators", "✗"), "lacks")
	assert.Contains(t, helpRow(sections, "This help", "↑/↓  j/k"), "mouse wheel")
	assert.Contains(t, helpRow(sections, "This help", "q / Ctrl+C"), "Close")
	assert.Contains(t, helpRow(sections, "This help", "Esc / ?"), "Close")
}

func TestHelpNamesEachKeyOnceASection(t *testing.T) {
	for title, rows := range helpSections(helpScreen(openHelp(120, 200))) {
		if strings.Contains(title, "Indicators") {
			continue // glyphs, not keys
		}
		seen := map[string]bool{}
		for _, row := range rows {
			column, _, _ := strings.Cut(row, "   ")
			for _, k := range strings.FieldsFunc(column, func(r rune) bool { return r == ' ' || r == '/' }) {
				assert.False(t, seen[k], "%s names %s twice", title, k)
				seen[k] = true
			}
		}
	}
}

func TestHelpAlignsItsDescriptions(t *testing.T) {
	sections := helpSections(helpScreen(openHelp(120, 200)))
	var columns []int
	for title, rows := range sections {
		if title == "" {
			continue
		}
		for _, row := range rows {
			column, _, _ := strings.Cut(row, "   ")
			description := strings.TrimLeft(row[len(column):], " ")
			columns = append(columns, lipgloss.Width(row)-lipgloss.Width(description))
		}
	}
	require.NotEmpty(t, columns)
	for _, c := range columns {
		assert.Equal(t, columns[0], c, "every description starts at the same column")
	}
}

func TestHelpPagesWithPgUpAndPgDn(t *testing.T) {
	app := openHelp(80, 24) // 20 rows of text between the padding and the footer
	top := helpScreen(app)

	paged := press(app, "pgdown")
	assert.Equal(t, helpScreen(press(app, slices.Repeat([]string{"down"}, 20)...)), helpScreen(paged),
		"PgDn scrolls one screen of text")
	assert.Equal(t, top, helpScreen(press(paged, "pgup")), "PgUp scrolls back one screen")
	assert.Equal(t, top, helpScreen(press(app, "pgup")), "PgUp at the top does nothing")
}

func TestHelpJumpsToTheEndsWithHomeAndEnd(t *testing.T) {
	app := openHelp(80, 24)
	top := helpScreen(app)

	end := press(app, "end")
	assert.Equal(t, helpScreen(press(app, slices.Repeat([]string{"down"}, scrollPastTheEnd)...)), helpScreen(end),
		"End shows the last row")
	assert.Equal(t, helpScreen(end), helpScreen(press(end, "pgdown")), "PgDn at the end does nothing")
	assert.Equal(t, top, helpScreen(press(end, "home")), "Home shows the first row")
}

func TestHelpNamesTheMouseWheelOnlyWithTheMouseOn(t *testing.T) {
	assert.Contains(t, helpScreen(openHelp(120, 200)), "mouse wheel")

	cfg := config.DefaultConfig()
	cfg.NoMouse = true
	assert.NotContains(t, helpScreen(openHelpWith(cfg, 120, 200)), "mouse wheel")
}

// helpTopRow is the first row of help text on screen, below the padding.
func helpTopRow(app App) string {
	return strings.TrimSpace(strings.Split(helpScreen(app), "\n")[helpPaddingHeight])
}

func TestHelpKeepsItsTopRowWhenTheWidthChanges(t *testing.T) {
	app := openHelp(120, 24)
	for range scrollPastTheEnd {
		if helpTopRow(app) == "Compare view" {
			break
		}
		app = press(app, "down")
	}
	require.Equal(t, "Compare view", helpTopRow(app))

	updated, _ := app.Update(tea.WindowSizeMsg{Width: MinTerminalWidth, Height: 24})

	assert.Equal(t, "Compare view", helpTopRow(updated.(App)),
		"rows that wrap above the top row must not shift what the screen shows")
}
