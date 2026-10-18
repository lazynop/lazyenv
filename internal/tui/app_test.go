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

func TestUpdateWindowSizeMsg(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Equal(t, 200, app.width)
	assert.Equal(t, 50, app.height)
	assert.True(t, app.ready)
}

func TestUpdateBackgroundColorMsg(t *testing.T) {
	app := newTestApp(nil)
	assert.True(t, app.hasDarkBg) // default

	// Simulate light background
	updated, cmd := app.Update(tea.BackgroundColorMsg{})
	app = updated.(App)
	assert.Nil(t, cmd)
	// The theme should have been rebuilt (we can't easily check IsDark() on a zero msg,
	// but we verify it doesn't crash and returns)
}

func TestUpdateFilesLoadedMsg(t *testing.T) {
	app := newTestApp(nil)
	f1 := makeTestFile(".env", "FOO")
	f2 := makeTestFile(".env.prod", "BAR")

	updated, cmd := app.Update(FilesLoadedMsg{
		Files: []*model.EnvFile{f1, f2},
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Len(t, app.fileList.Files, 2)
	assert.Equal(t, f1, app.varList.File)
}

func TestUpdateFilesLoadedMsgError(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(FilesLoadedMsg{
		Err: assert.AnError,
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Contains(t, app.statusBar.Message, "Error")
}

func TestUpdateFilesLoadedMsgErrorRendersInErrorColor(t *testing.T) {
	app := newTestApp(nil)

	updated, _ := app.Update(FilesLoadedMsg{
		Err: assert.AnError,
	})
	app = updated.(App)

	view := statusView(app)
	assertStatusColor(t, view, statusErrorSGR)
}

func TestUpdateFilesLoadedMsgEmpty(t *testing.T) {
	app := newTestApp(nil)

	updated, cmd := app.Update(FilesLoadedMsg{
		Files: []*model.EnvFile{},
	})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Empty(t, app.fileList.Files)
}

func TestUpdateClearMessageMsg(t *testing.T) {
	app := newTestApp(nil)
	app.statusBar.SetMessage("temporary")
	assert.Equal(t, "temporary", app.statusBar.Message)

	updated, cmd := app.Update(ClearMessageMsg{gen: app.statusBar.msgGen})
	app = updated.(App)

	assert.Nil(t, cmd)
	assert.Equal(t, "", app.statusBar.Message)
}

func TestUpdateClearMessageMsgIgnoresStaleGen(t *testing.T) {
	app := newTestApp(nil)
	app.statusBar.SetMessage("first")
	staleGen := app.statusBar.msgGen
	app.statusBar.SetMessage("second") // newer message bumps the generation

	updated, _ := app.Update(ClearMessageMsg{gen: staleGen})
	app = updated.(App)

	assert.Equal(t, "second", app.statusBar.Message,
		"a clear scheduled for an earlier message must not wipe a newer one")
}

func TestNavigationUpDownInFilePanel(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	f3 := makeTestFile(".env.local", "C")
	app := newTestApp([]*model.EnvFile{f1, f2, f3})
	app.focus = FocusFiles

	// Move down
	updated, _ := app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.fileList.Cursor)
	assert.Equal(t, f2, app.varList.File, "var panel should sync with file cursor")

	// Move up
	updated, _ = app.Update(keyMsg("k"))
	app = updated.(App)
	assert.Equal(t, 0, app.fileList.Cursor)
	assert.Equal(t, f1, app.varList.File)
}

func TestNavigationUpDownInVarPanel(t *testing.T) {
	f := makeTestFile(".env", "A", "B", "C")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("j"))
	app = updated.(App)
	assert.Equal(t, 1, app.varList.Cursor)

	updated, _ = app.Update(keyMsg("k"))
	app = updated.(App)
	assert.Equal(t, 0, app.varList.Cursor)
}

func TestEnterSelectsFileAndSwitchesToVars(t *testing.T) {
	f1 := makeTestFile(".env", "A")
	f2 := makeTestFile(".env.prod", "B")
	app := newTestApp([]*model.EnvFile{f1, f2})
	app.focus = FocusFiles

	// Move to second file and press enter
	updated, _ := app.Update(keyMsg("j"))
	app = updated.(App)
	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)

	assert.Equal(t, FocusVars, app.focus)
	assert.Equal(t, f2, app.varList.File)
}

func TestQuitReturnsQuitCmd(t *testing.T) {
	app := newTestApp(nil)

	_, cmd := app.Update(keyMsg("q"))
	// tea.Quit is a function, we can check cmd is not nil
	assert.NotNil(t, cmd)
}

func TestViewNotReady(t *testing.T) {
	app := NewApp(config.DefaultConfig(), StartupMessages{})
	// ready is false by default
	view := app.View()
	assert.Contains(t, view.Content, "Loading")
}

func TestInitReturnsBatch(t *testing.T) {
	app := newTestApp(nil)

	// Init builds the initial command batch (scan dir + request bg color);
	// we only assert it is non-nil without running the closures.
	assert.NotNil(t, app.Init())
}

// startApp runs the commands Init batches and feeds their messages to Update,
// as the Bubble Tea runtime does at startup. cfg.Dir is scanned, so callers
// point it at an empty directory.
func startApp(t *testing.T, app App) App {
	t.Helper()
	batch, ok := app.Init()().(tea.BatchMsg)
	require.True(t, ok, "Init must batch its startup commands")
	app.width, app.height, app.ready = 120, 40, true
	for _, cmd := range batch {
		updated, _ := app.Update(cmd())
		app = updated.(App)
	}
	return app
}

func TestStartupNoticeFlashesWithoutBlocking(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()
	notice := `unknown theme "does-not-exist" (see --list-themes)`

	app := startApp(t, NewApp(cfg, StartupMessages{Notices: []string{notice}}))

	assert.Equal(t, ModeNormal, app.mode, "a notice must not open the blocking config error screen")
	assert.Equal(t, notice, app.statusBar.Message)
	_, cmd := app.Update(keyMsg("j"))
	assert.Nil(t, cmd, "the next key must reach the TUI instead of quitting")
}

func TestStartupNoticeRendersInErrorColor(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()

	app := startApp(t, NewApp(cfg, StartupMessages{Notices: []string{`unknown theme "does-not-exist"`}}))

	view := statusView(app)
	assertStatusColor(t, view, statusErrorSGR)
}

func TestConfigErrorBlocksAndQuitsOnNextKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()

	app := startApp(t, NewApp(cfg, StartupMessages{ConfigErrors: []string{`invalid sort value: "random"`}}))

	assert.Equal(t, ModeConfigError, app.mode)
	assert.Contains(t, app.View().Content, `invalid sort value: "random"`)
	_, cmd := app.Update(keyMsg("j"))
	require.NotNil(t, cmd)
	assert.IsType(t, tea.QuitMsg{}, cmd(), "a rejected config file must quit on the next key")
}

func TestConfigErrorScreenNamesTheFile(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()

	app := startApp(t, NewApp(cfg, StartupMessages{
		ConfigPath:   "/home/user/.lazyenvrc",
		ConfigErrors: []string{`invalid sort value: "random"`},
	}))

	assert.Contains(t, app.View().Content, "/home/user/.lazyenvrc")
}

func TestConfigErrorScreenWithoutAPathLeavesNoGap(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()

	app := startApp(t, NewApp(cfg, StartupMessages{ConfigErrors: []string{`invalid sort value: "random"`}}))

	var rows []string
	for _, row := range strings.Split(stripAnsi(app.View().Content), "\n") {
		rows = append(rows, strings.Trim(row, " │╭╮╰╯─"))
	}
	title := slices.Index(rows, "Configuration Error")
	require.NotEqual(t, -1, title)
	assert.Equal(t, []string{"", `invalid sort value: "random"`}, rows[title+1:title+3],
		"the error follows the title after one blank row, not a blank path row")
}

func TestConfigErrorScreenCutsWhatDoesNotFit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()
	errs := make([]string, 6)
	for i := range errs {
		errs[i] = fmt.Sprintf("invalid colors.color%d: \"teal\" (expected hex like #RRGGBB or ANSI 0-255)", i)
	}
	app := NewApp(cfg, StartupMessages{ConfigPath: ".lazyenvrc", ConfigErrors: errs})

	updated, _ := app.Update(tea.WindowSizeMsg{Width: MinTerminalWidth, Height: MinTerminalHeight})
	view := stripAnsi(updated.(App).View().Content)

	assert.LessOrEqual(t, lipgloss.Height(view), MinTerminalHeight)
	assert.Contains(t, view, "…", "the cut errors end in an ellipsis")
	assert.Contains(t, view, "see lazyenv --check-config", "the hint points to the full list")
	assert.NotContains(t, view, "color5", "the last errors are cut")
}

func TestConfigErrorScreenAlsoShowsNotices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Dir = t.TempDir()

	app := startApp(t, NewApp(cfg, StartupMessages{
		ConfigErrors: []string{`invalid sort value: "random"`},
		Notices:      []string{`unknown theme "nope"`},
	}))

	assert.Equal(t, ModeConfigError, app.mode)
	assert.Contains(t, app.View().Content, `unknown theme "nope"`,
		"the run stops on this screen, so a notice must not be lost behind it")
}

func TestFileListWidthBelowMinimumIsRaised(t *testing.T) {
	app := newTestApp(nil)
	app.config.Layout.FileListWidth = 5

	updated, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Equal(t, 20, updated.(App).fileList.Width)
}
