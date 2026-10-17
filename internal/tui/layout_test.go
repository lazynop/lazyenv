package tui

import (
	"flag"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

var updateGolden = flag.Bool("update", false, "rewrite "+viewsGoldenPath+" from the current renders")

const viewsGoldenPath = "testdata/views.golden"

// layoutSecrets are the secret values of layoutFiles: none may be on screen
// while secrets are masked. A leak shows only what its column fits, so each
// must be short enough for every column to draw it whole, which
// TestLayoutSecretsAreDrawnWhole checks.
var layoutSecrets = []string{"hunter2", "ghp_x7Kq9", "stg-pass"}

// layoutFiles returns two files whose secrets are detected by key name
// (API_TOKEN) and by value (GREETING), with more vars than any panel shows.
func layoutFiles() []*model.EnvFile {
	var env strings.Builder
	env.WriteString("API_TOKEN=hunter2\nGREETING=ghp_x7Kq9\nHOST=localhost\nEMPTY=\n")
	for i := range 30 {
		fmt.Fprintf(&env, "VAR_%02d=value_%02d\n", i, i)
	}
	local := "API_TOKEN=stg-pass\nHOST=example.com\nVAR_00=value_00\nONLY_LOCAL=1\n"
	return []*model.EnvFile{
		parser.ParseBytes(".env", []byte(env.String()), config.SecretsConfig{}),
		parser.ParseBytes(".env.local", []byte(local), config.SecretsConfig{}),
	}
}

// viewScenario renders one screen of the TUI at a terminal size, with secrets
// revealed or masked.
type viewScenario struct {
	name   string
	mode   AppMode // -1 for a screen outside App
	render func(t *testing.T, width, height int, reveal bool) string
}

// appScenario sizes a fresh App over layoutFiles, drives it with setup and
// renders it, failing unless setup reached mode.
func appScenario(name string, mode AppMode, setup func(app App) App) viewScenario {
	return viewScenario{name, mode, func(t *testing.T, width, height int, reveal bool) string {
		t.Helper()
		updated, _ := newTestApp(layoutFiles()).Update(tea.WindowSizeMsg{Width: width, Height: height})
		app := updated.(App)
		app.varList.SetShowSecrets(reveal)
		app = setup(app)
		require.Equal(t, mode, app.mode, "%s: setup should reach the mode", name)
		return app.View().Content
	}}
}

// press sends keys to app: named keys (enter, esc, tab, arrows) by code, the
// rest as text.
func press(app App, keys ...string) App {
	codes := map[string]rune{"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab,
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight}
	for _, k := range keys {
		msg := tea.KeyPressMsg{Text: k}
		if code, ok := codes[k]; ok {
			msg = tea.KeyPressMsg{Code: code}
		}
		updated, _ := app.Update(msg)
		app = updated.(App)
	}
	return app
}

// onVars moves the focus to the var list, the way the user does.
func onVars(app App) App {
	return press(app, "right")
}

// viewScenarios covers every AppMode and the theme preview. A new mode or
// screen gets a scenario here; TestViewScenariosCoverEveryMode checks modes.
var viewScenarios = []viewScenario{
	appScenario("normal", ModeNormal, func(app App) App { return app }),
	appScenario("vars", ModeNormal, onVars),
	appScenario("peek edited secret", ModeNormal, func(app App) App {
		app.fileList.Files[0].UpdateVar(0, "edited", true) // API_TOKEN
		app.varList.Refresh()
		return press(onVars(app), "p")
	}),
	appScenario("peek secret renamed to a plain key", ModeNormal, func(app App) App {
		app = onVars(app)
		app.fileList.Files[0].UpdateVar(0, "x", true) // API_TOKEN
		app.mode = ModeEditing
		app.editor.StartEditKey(&app.fileList.Files[0].Vars[0], 0)
		app.editor.input.SetValue("PLAIN")
		return press(app, "enter", "p")
	}),
	appScenario("peek secret value renamed", ModeNormal, func(app App) App {
		app = press(onVars(app), "j", "E") // GREETING, a secret by its value only
		app.editor.input.SetValue("WELCOME")
		return press(app, "enter", "p")
	}),
	appScenario("peek secret re-added with a plain value", ModeNormal, func(app App) App {
		f := app.fileList.Files[0]
		f.DeleteVar(1)
		f.AddVar("GREETING", "hello", false)
		app.varList.Refresh()
		app = onVars(app)
		app.varList.SetCursor(len(f.Vars) - 1)
		return press(app, "p")
	}),
	appScenario("search", ModeSearching, func(app App) App { return press(onVars(app), "/", "h", "u", "n") }),
	appScenario("edit value", ModeEditing, func(app App) App { return press(onVars(app), "j", "j", "e") }),
	appScenario("edit long value", ModeEditing, func(app App) App {
		app = press(onVars(app), "j", "j", "e")
		app.editor.input.SetValue(strings.Repeat("long-", 40) + "end")
		app.editor.input.CursorEnd()
		return app
	}),
	appScenario("confirm delete var", ModeConfirmDelete, func(app App) App { return press(onVars(app), "d") }),
	appScenario("reorder menu", ModeReorderMenu, func(app App) App { return press(onVars(app), "O") }),
	appScenario("reorder confirm", ModeReorderConfirm, func(app App) App { return press(onVars(app), "O", "a") }),
	appScenario("help", ModeHelp, func(app App) App { return press(app, "?") }),
	appScenario("new file", ModeCreateFile, func(app App) App { return press(app, "N") }),
	appScenario("duplicate file", ModeDuplicateFile, func(app App) App { return press(app, "C") }),
	appScenario("rename file", ModeRenameFile, func(app App) App { return press(app, "R") }),
	appScenario("template file", ModeTemplateFile, func(app App) App { return press(app, "T") }),
	appScenario("confirm delete file", ModeConfirmDeleteFile, func(app App) App { return press(app, "D") }),
	appScenario("compare select", ModeCompareSelect, func(app App) App { return press(app, "c") }),
	appScenario("compare", ModeComparing, func(app App) App { return press(app, "c", "j", "enter") }),
	appScenario("compare edit", ModeEditingCompare, func(app App) App {
		return press(app, "c", "j", "enter", "j", "j", "e") // HOST, third in both files
	}),
	appScenario("matrix", ModeMatrix, func(app App) App { return press(app, "m") }),
	appScenario("matrix add", ModeMatrixEditing, func(app App) App {
		return press(app, "m", "j", "right", "a") // EMPTY, missing from .env.local
	}),
	appScenario("matrix confirm delete", ModeConfirmMatrixDelete, func(app App) App { return press(app, "m", "d") }),
	{"config error", ModeConfigError, func(t *testing.T, width, height int, _ bool) string {
		app := NewApp(config.DefaultConfig(), StartupMessages{ConfigErrors: []string{`invalid sort value: "random"`}})
		updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
		require.Equal(t, ModeConfigError, updated.(App).mode)
		return updated.(App).View().Content
	}},
	{"theme preview", -1, func(t *testing.T, width, height int, _ bool) string {
		m := NewThemePreview(true)
		m.searchPaths = []string{".lazyenvrc", "/home/user/.config/lazyenv/config.toml", "/home/user/.lazyenvrc"}
		updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
		return updated.(ThemePreviewModel).View().Content
	}},
}

// TestViewScenariosCoverEveryMode fails on an AppMode with no scenario in
// viewScenarios. It reads the modes from the iota block in app.go, where a
// mode's value is its position.
func TestViewScenariosCoverEveryMode(t *testing.T) {
	covered := map[AppMode]bool{}
	for _, sc := range viewScenarios {
		covered[sc.mode] = true
	}
	file, err := goparser.ParseFile(token.NewFileSet(), "app.go", nil, goparser.SkipObjectResolution)
	require.NoError(t, err)
	var modes []string
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.CONST {
			if first := gen.Specs[0].(*ast.ValueSpec); first.Names[0].Name == "ModeNormal" {
				for _, spec := range gen.Specs {
					modes = append(modes, spec.(*ast.ValueSpec).Names[0].Name)
				}
			}
		}
	}
	require.NotEmpty(t, modes, "the AppMode const block starting at ModeNormal should be in app.go")
	for value, name := range modes {
		assert.True(t, covered[AppMode(value)], "%s has no scenario in viewScenarios", name)
	}
}

// knownTooTall lists the scenarios that still outgrow the terminal, each with
// the issue tracking it. An entry whose scenario fits everywhere fails the
// test: remove it.
var knownTooTall = map[string]string{
	"help": "#57",
}

// gridWidths and gridHeights are the terminal sizes every scenario renders at.
// The grid starts at the minimum terminal size, below which every screen gives
// way to tooSmallView (TestTooSmallTerminalShowsANotice).
var (
	gridWidths  = []int{MinTerminalWidth, 80, 100, 120}
	gridHeights = []int{MinTerminalHeight, 12, 16, 24, 40}
)

// TestViewsFitTheTerminalAndHideSecrets renders every scenario across the grid
// of terminal sizes.
func TestViewsFitTheTerminalAndHideSecrets(t *testing.T) {
	for _, sc := range viewScenarios {
		_, known := knownTooTall[sc.name]
		tooTall := false
		for _, width := range gridWidths {
			for _, height := range gridHeights {
				view := sc.render(t, width, height, false)
				at := fmt.Sprintf("%s at %dx%d", sc.name, width, height)
				if lipgloss.Height(view) > height {
					tooTall = true
					assert.True(t, known, "%s: taller than the terminal", at)
				}
				assert.LessOrEqual(t, lipgloss.Width(view), width, "%s: wider than the terminal", at)
				for _, secret := range layoutSecrets {
					assert.NotContains(t, stripAnsi(view), secret, "%s: a masked secret is on screen", at)
				}
			}
		}
		if known {
			assert.True(t, tooTall, "%s now fits the terminal: remove it from knownTooTall", sc.name)
		}
	}
}

// TestLayoutSecretsAreDrawnWhole renders every scenario across the grid with
// secrets revealed. A column that draws a secret cut (its start, then "..")
// would show only that start on a leak, which the check for the whole secret
// in TestViewsFitTheTerminalAndHideSecrets misses. Each secret must also be
// drawn somewhere, or that check could not catch its leak at all.
func TestLayoutSecretsAreDrawnWhole(t *testing.T) {
	drawn := map[string]bool{}
	for _, sc := range viewScenarios {
		for _, width := range gridWidths {
			for _, height := range gridHeights {
				view := stripAnsi(sc.render(t, width, height, true))
				for _, secret := range layoutSecrets {
					drawn[secret] = drawn[secret] || strings.Contains(view, secret)
					for n := 1; n < len(secret); n++ {
						assert.NotContains(t, view, secret[:n]+"..", "%s at %dx%d: %q is drawn cut, shorten it",
							sc.name, width, height, secret)
					}
				}
			}
		}
	}
	for _, secret := range layoutSecrets {
		assert.True(t, drawn[secret], "%q is never drawn, even revealed", secret)
	}
}

// plainRows returns view without colors or the padding at the end of its rows.
func plainRows(view string) string {
	rows := strings.Split(stripAnsi(view), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return strings.Join(rows, "\n")
}

// TestViewsMatchGolden compares every scenario's plain-text render at 80x24
// with testdata/views.golden, so a change to any screen shows up as a diff.
// With -update it rewrites the file and still fails: the rewrite is only
// done once its diff has been read.
func TestViewsMatchGolden(t *testing.T) {
	var got strings.Builder
	for _, sc := range viewScenarios {
		fmt.Fprintf(&got, "=== %s\n%s\n", sc.name, plainRows(sc.render(t, 80, 24, false)))
	}
	if *updateGolden {
		require.NoError(t, os.MkdirAll("testdata", 0o755))
		require.NoError(t, os.WriteFile(viewsGoldenPath, []byte(got.String()), 0o644))
		t.Fatalf("rewrote internal/tui/%s: read git diff on it, check each changed screen is intended, "+
			"name them on a Screens: line of the commit message, then rerun without -update", viewsGoldenPath)
	}
	want, err := os.ReadFile(viewsGoldenPath)
	require.NoError(t, err, "create %s with -update", viewsGoldenPath)
	assert.Equal(t, string(want), got.String(), "a screen changed: when the change is intended, "+
		"rewrite internal/tui/%s with go test ./internal/tui -run TestViewsMatchGolden -update", viewsGoldenPath)
}
