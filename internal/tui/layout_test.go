package tui

import (
	"flag"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
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

var layoutSecrets = []string{"hunter2", "ghp_x7Kq9", "stg-pass"}

func layoutFiles() []*model.EnvFile {
	return layoutFilesNamed(".env", ".env.local")
}

const apiTokenIndex = 0

func layoutFilesNamed(env, local string) []*model.EnvFile {
	const secretByKeyName, secretByValue = "API_TOKEN=hunter2\n", "GREETING=ghp_x7Kq9\n"
	const morePlainVarsThanAnyPanelShows = 30
	var vars strings.Builder
	vars.WriteString(secretByKeyName + secretByValue + "HOST=localhost\nEMPTY=\n")
	for i := range morePlainVarsThanAnyPanelShows {
		fmt.Fprintf(&vars, "VAR_%02d=value_%02d\n", i, i)
	}
	localVars := "API_TOKEN=stg-pass\nHOST=example.com\nVAR_00=value_00\nONLY_LOCAL=1\n"
	return []*model.EnvFile{
		parser.ParseBytes(env, []byte(vars.String()), config.SecretsConfig{}),
		parser.ParseBytes(local, []byte(localVars), config.SecretsConfig{}),
	}
}

const screenOutsideApp = -1

type viewScenario struct {
	name  string
	mode  AppMode
	setup func(app App) App
	draw  func(t *testing.T, width, height int) string
}

func appScenario(name string, mode AppMode, setup func(app App) App) viewScenario {
	return viewScenario{name: name, mode: mode, setup: setup}
}

func (sc viewScenario) render(t *testing.T, width, height int, reveal bool) string {
	t.Helper()
	if sc.setup == nil {
		return sc.draw(t, width, height)
	}
	return sc.renderOver(t, layoutFiles(), width, height, reveal)
}

func (sc viewScenario) renderOver(t *testing.T, files []*model.EnvFile, width, height int, reveal bool) string {
	t.Helper()
	app := resize(newTestApp(files), width, height)
	app.varList.SetShowSecrets(reveal)
	app = sc.setup(app)
	require.Equal(t, sc.mode, app.mode, "%s: setup should reach the mode", sc.name)
	return app.View().Content
}

func configErrorScenario(name string, startup StartupMessages) viewScenario {
	return viewScenario{name: name, mode: ModeConfigError, draw: func(t *testing.T, width, height int) string {
		t.Helper()
		app := resize(NewApp(config.DefaultConfig(), startup), width, height)
		require.Equal(t, ModeConfigError, app.mode, "%s: startup should reach the mode", name)
		return app.View().Content
	}}
}

func press(app App, keys ...string) App {
	for _, k := range keys {
		updated, _ := app.Update(keyMsg(k))
		app = updated.(App)
	}
	return app
}

func typing(text string) []string {
	return strings.Split(text, "")
}

func writeAndParse(t *testing.T, path, content string) *model.EnvFile {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	f, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)
	return f
}

func blockBackup(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(path+".bak", "blocker"), 0o755))
}

func newCompareAppOnDisk(t *testing.T) (app App, pathA, pathB string) {
	t.Helper()
	pathA = filepath.Join(t.TempDir(), ".env")
	pathB = filepath.Join(t.TempDir(), ".env.prod")
	fA := writeAndParse(t, pathA, "FOO=a\n")
	fB := writeAndParse(t, pathB, "FOO=b\n")

	app = enterCompareMode(t, fA, fB)
	require.Same(t, fA, app.diffView.FileA)
	require.Same(t, fB, app.diffView.FileB)
	return app, pathA, pathB
}

func replaceInput(openKey, text string) []string {
	return slices.Concat([]string{openKey, "ctrl+u"}, typing(text), []string{"enter"})
}

func assertQuits(t *testing.T, cmd tea.Cmd, msgAndArgs ...any) {
	t.Helper()
	require.NotNil(t, cmd, msgAndArgs...)
	assert.IsType(t, tea.QuitMsg{}, cmd(), msgAndArgs...)
}

func down(n int) []string {
	return slices.Repeat([]string{"j"}, n)
}

func resize(app App, width, height int) App {
	updated, _ := app.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return updated.(App)
}

func onVars(app App) App {
	return press(app, "right")
}

var viewScenarios = []viewScenario{
	appScenario("normal", ModeNormal, func(app App) App { return app }),
	appScenario("vars", ModeNormal, onVars),
	appScenario("peek edited secret", ModeNormal, func(app App) App {
		app.fileList.Files[0].UpdateVar(apiTokenIndex, "edited")
		app.varList.Refresh()
		return press(onVars(app), "p")
	}),
	appScenario("peek secret renamed to a plain key", ModeNormal, func(app App) App {
		app = onVars(app)
		app.fileList.Files[0].UpdateVar(apiTokenIndex, "x")
		app.mode = ModeEditing
		app.editor.StartEditKey(&app.fileList.Files[0].Vars[apiTokenIndex], apiTokenIndex)
		app.editor.input.SetValue("PLAIN")
		return press(app, "enter", "p")
	}),
	appScenario("peek secret value renamed", ModeNormal, func(app App) App {
		editingGreetingKey := press(onVars(app), "j", "E")
		editingGreetingKey.editor.input.SetValue("WELCOME")
		return press(editingGreetingKey, "enter", "p")
	}),
	appScenario("peek secret re-added with a plain value", ModeNormal, func(app App) App {
		f := app.fileList.Files[0]
		f.DeleteVar(1)
		f.AddVar("GREETING", "hello")
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
	appScenario("help scrolled to the end", ModeHelp, func(app App) App {
		return press(app, "?", "end")
	}),
	appScenario("new file", ModeCreateFile, func(app App) App { return press(app, "N") }),
	appScenario("duplicate file", ModeDuplicateFile, func(app App) App { return press(app, "C") }),
	appScenario("rename file", ModeRenameFile, func(app App) App { return press(app, "R") }),
	appScenario("template file", ModeTemplateFile, func(app App) App { return press(app, "T") }),
	appScenario("confirm delete file", ModeConfirmDeleteFile, func(app App) App { return press(app, "D") }),
	appScenario("compare select", ModeCompareSelect, func(app App) App { return press(app, "c") }),
	appScenario("compare", ModeComparing, func(app App) App { return press(app, "c", "j", "enter") }),
	appScenario("compare edit", ModeEditingCompare, func(app App) App {
		editingHostInBothFiles := press(app, "c", "j", "enter", "j", "j", "e")
		return editingHostInBothFiles
	}),
	appScenario("matrix", ModeMatrix, func(app App) App { return press(app, "m") }),
	appScenario("matrix add", ModeMatrixEditing, func(app App) App {
		addingEmptyToEnvLocal := press(app, "m", "j", "right", "a")
		return addingEmptyToEnvLocal
	}),
	appScenario("matrix confirm delete", ModeConfirmMatrixDelete, func(app App) App { return press(app, "m", "d") }),
	configErrorScenario("config error", StartupMessages{
		ConfigPath:   ".lazyenvrc",
		ConfigErrors: []string{`invalid sort value: "random"`},
	}),
	configErrorScenario("config error, long path", StartupMessages{
		ConfigPath:   "/home/user/projects/a-rather-long-project-name/configuration/lazyenv/config.toml",
		ConfigErrors: []string{`invalid sort value: "random"`, `invalid colors.primary: "teal" (expected hex like #RRGGBB or ANSI 0-255)`},
	}),
	{name: "theme preview", mode: screenOutsideApp, draw: func(t *testing.T, width, height int) string {
		m := NewThemePreview(config.DefaultConfig())
		m.searchPaths = []string{".lazyenvrc", "/home/user/.config/lazyenv/config.toml", "/home/user/.lazyenvrc"}
		updated, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
		return updated.(ThemePreviewModel).View().Content
	}},
}

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
	for iotaPosition, name := range modes {
		assert.True(t, covered[AppMode(iotaPosition)], "%s has no scenario in viewScenarios", name)
	}
}

var tooTallIssueByScenario = map[string]string{}

var (
	gridWidths  = []int{MinTerminalWidth, 80, 100, 120}
	gridHeights = []int{MinTerminalHeight, 12, 16, 24, 40}
)

func TestViewsFitTheTerminalAndHideSecrets(t *testing.T) {
	for _, sc := range viewScenarios {
		_, known := tooTallIssueByScenario[sc.name]
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
			assert.True(t, tooTall, "%s now fits the terminal: remove it from tooTallIssueByScenario", sc.name)
		}
	}
}

func TestLayoutSecretsAreDrawnWholeSoTheLeakCheckCanCatchThem(t *testing.T) {
	drawn := map[string]bool{}
	for _, sc := range viewScenarios {
		for _, width := range gridWidths {
			for _, height := range gridHeights {
				view := stripAnsi(sc.render(t, width, height, true))
				for _, secret := range layoutSecrets {
					drawn[secret] = drawn[secret] || strings.Contains(view, secret)
					for n := 1; n < len(secret); n++ {
						assert.NotContains(t, view, secret[:n]+"..", "%s at %dx%d: %q is drawn cut, so a leak would show only its start, "+
							"which the check for the whole secret misses: shorten it",
							sc.name, width, height, secret)
					}
				}
			}
		}
	}
	for _, secret := range layoutSecrets {
		assert.True(t, drawn[secret], "%q is never drawn, even revealed, so no leak check can catch its leak", secret)
	}
}

func TestFileNamesAreDrawnFlat(t *testing.T) {
	const raw, flat = "\n\x1b]0;x\x07", "↵␛]0;x␇"
	prefillSanitizedByTheInput := map[AppMode]bool{ModeDuplicateFile: true, ModeRenameFile: true, ModeTemplateFile: true}
	for _, sc := range viewScenarios {
		if sc.setup == nil {
			continue
		}
		for _, width := range gridWidths {
			for _, height := range gridHeights {
				at := fmt.Sprintf("%s at %dx%d", sc.name, width, height)
				got := sc.renderOver(t, layoutFilesNamed(".env"+raw, ".env.local"+raw), width, height, false)
				assert.NotContains(t, got, "\x1b]", "%s: a file name sent the terminal a sequence", at)
				if !prefillSanitizedByTheInput[sc.mode] {
					want := sc.renderOver(t, layoutFilesNamed(".env"+flat, ".env.local"+flat), width, height, false)
					assert.Equal(t, want, got, "%s: a file name is drawn unflattened", at)
				}
			}
		}
	}
}

func plainRows(view string) string {
	rows := strings.Split(stripAnsi(view), "\n")
	for i := range rows {
		rows[i] = strings.TrimRight(rows[i], " ")
	}
	return strings.Join(rows, "\n")
}

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
