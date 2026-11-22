package tui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
	"github.com/lazynop/lazyenv/internal/util"
)

func TestEditExistingVarConfirm(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("e"))
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	app.editor.input.SetValue("new_value")

	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "new_value", f.Vars[0].Value)
	assert.True(t, f.Modified)
}

func TestEditMasksTheNewValueWhenItIsASecret(t *testing.T) {
	tests := []struct {
		name, line, newValue string
		masked               bool
	}{
		{"plain value edited to a token", "GREETING=hello", "ghp_x7Kq9", true},
		{"token edited to a plain value", "GREETING=ghp_x7Kq9", "hello", false},
		{"secret key edited to a plain value", "API_TOKEN=hunter2", "hello", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parser.ParseBytes(".env", []byte(tt.line+"\n"), config.SecretsConfig{})
			app := press(resize(newTestApp([]*model.EnvFile{f}), 80, 24), "right", "e")
			app.editor.input.SetValue(tt.newValue)

			view := stripAnsi(press(app, "enter").View().Content)

			assertMaskedInView(t, view, tt.newValue, tt.masked)
		})
	}
}

func assertMaskedInView(t *testing.T, view, value string, masked bool) {
	t.Helper()
	if masked {
		assert.NotContains(t, view, value, "a secret value must be masked")
		assert.Contains(t, view, util.MaskValue(value))
	} else {
		assert.Contains(t, view, value, "a plain value must be shown")
	}
}

var valueEdits = []struct {
	name   string
	keys   []string
	label  string
	mode   AppMode // the mode the keys leave the app in when the bar stays closed
	edited int     // index of the file whose value the keys edit
}{
	{"var list e", []string{"right", "e"}, "Edit CERT: ", ModeNormal, 0},
	{"compare e", []string{"c", "j", "enter", "e"}, "Edit CERT in .env: ", ModeComparing, 0},
	{"compare E", []string{"c", "j", "enter", "E"}, "Edit CERT in .env.prod: ", ModeComparing, 1},
}

func valueEditApp(edited int, line string) (App, []*model.EnvFile) {
	lines := []string{"CERT=plain", "CERT=plain"}
	lines[edited] = line
	files := []*model.EnvFile{
		parser.ParseBytes(".env", []byte(lines[0]+"\n"), config.SecretsConfig{}),
		parser.ParseBytes(".env.prod", []byte(lines[1]+"\n"), config.SecretsConfig{}),
	}
	return resize(newTestApp(files), 120, 40), files
}

func TestConfirmingAnUntouchedEditKeepsTheValue(t *testing.T) {
	for _, edit := range valueEdits {
		t.Run(edit.name, func(t *testing.T) {
			app, files := valueEditApp(edit.edited, "CERT=plain")

			press(press(app, edit.keys...), "enter")

			assert.Equal(t, "plain", files[edit.edited].Vars[0].Value)
			assert.False(t, files[edit.edited].Modified, "an untouched edit must not modify the file")
		})
	}
}

func TestEditOpensTheBarOnAPlainValue(t *testing.T) {
	for _, edit := range valueEdits {
		t.Run(edit.name, func(t *testing.T) {
			app, files := valueEditApp(edit.edited, "CERT=plain")
			app = press(app, edit.keys...)
			require.Contains(t, stripAnsi(app.View().Content), edit.label+"> plain")

			view := stripAnsi(press(app, "z", "enter").View().Content)

			assert.Contains(t, view, "Modified CERT")
			assert.Contains(t, view, "plainz")
			assert.Equal(t, "plainz", files[edit.edited].Vars[0].Value)
			assert.True(t, files[edit.edited].Modified)
		})
	}
}

func TestEditRefusesAValueTheBarCannotShow(t *testing.T) {
	values := []struct{ name, line, value string }{
		{"line break", `CERT="line1\nline2"`, "line1\nline2"},
		{"tab", `CERT="a\tb"`, "a\tb"},
		{"escape", "CERT=a\x1bb", "a\x1bb"},
		{"invalid byte", "CERT=a\xffb", "a\xffb"},
	}
	for _, v := range values {
		for _, edit := range valueEdits {
			t.Run(v.name+"/"+edit.name, func(t *testing.T) {
				app, files := valueEditApp(edit.edited, v.line)
				app = press(app, edit.keys...)

				view := stripAnsi(app.View().Content)
				assert.Equal(t, edit.mode, app.mode, "the edit bar must not open")
				assert.NotContains(t, view, "Edit CERT")
				assert.Contains(t, view, "Value holds line breaks or control characters: can't edit it here")

				press(app, "z", "enter")

				assert.Equal(t, v.value, files[edit.edited].Vars[0].Value)
				for _, f := range files {
					assert.False(t, f.Modified, "a refused edit must not modify %s", f.Name)
				}
			})
		}
	}
}

// TestEditingALongValueKeepsAllOfIt edits a value longer than any input limit
// would allow, appending one character: the rest of the value survives.
func TestEditingALongValueKeepsAllOfIt(t *testing.T) {
	long := strings.Repeat("a", 1500)
	f := parser.ParseBytes(".env", []byte("CERT="+long+"\n"), config.SecretsConfig{})
	app := resize(newTestApp([]*model.EnvFile{f}), 80, 24)

	press(app, "right", "e", "z", "enter")

	assert.Equal(t, long+"z", f.Vars[0].Value)
}

func TestEditExistingVarCancel(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("e"))
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	app.editor.input.SetValue("changed")

	updated, _ = app.Update(keyMsg("esc"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "val_FOO", f.Vars[0].Value, "value should be unchanged after cancel")
}

func TestAddVariableTwoStepFlow(t *testing.T) {
	f := makeTestFile(".env", "EXISTING")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("a"))
	app = updated.(App)
	assert.Equal(t, ModeEditing, app.mode)

	app.editor.input.SetValue("NEW_KEY")
	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)

	assert.Equal(t, ModeEditing, app.mode)
	assert.Equal(t, addStepValue, app.editor.addStep)

	app.editor.input.SetValue("new_value")
	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	require.Len(t, f.Vars, 2)
	assert.Equal(t, "NEW_KEY", f.Vars[1].Key)
	assert.Equal(t, "new_value", f.Vars[1].Value)
}

func startAddWithKey(t *testing.T, f *model.EnvFile, key string) App {
	t.Helper()
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars
	updated, _ := app.Update(keyMsg("a"))
	app = updated.(App)
	require.Equal(t, ModeEditing, app.mode)
	app.editor.input.SetValue(key)
	updated, _ = app.Update(keyMsg("enter"))
	return updated.(App)
}

func TestAddVariableRejectsAKeyTheParserWouldDropOnReload(t *testing.T) {
	for _, key := range []string{"has space", "CAFÉ", "日本", "1ST"} {
		f := makeTestFile(".env", "EXISTING")
		app := startAddWithKey(t, f, key)

		assert.Equal(t, ModeNormal, app.mode, key)
		assert.Contains(t, app.statusBar.Message, "Invalid key", key)
		assert.Len(t, f.Vars, 1, key)
	}
}

func TestAddVariableRejectsAnExistingKey(t *testing.T) {
	f := makeTestFile(".env", "EXISTING")
	app := startAddWithKey(t, f, "EXISTING")

	assert.Equal(t, ModeNormal, app.mode)
	assert.Contains(t, app.statusBar.Message, "already exists")
	assert.Len(t, f.Vars, 1)
}

func TestAddVariableEmptyKeyNoOp(t *testing.T) {
	f := makeTestFile(".env", "EXISTING")
	app := startAddWithKey(t, f, "   ")

	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.statusBar.Message)
	assert.Len(t, f.Vars, 1)
}

func TestAddVariableTrimsTheKey(t *testing.T) {
	f := makeTestFile(".env", "EXISTING")
	app := startAddWithKey(t, f, "  NEW_KEY ")
	require.Equal(t, addStepValue, app.editor.addStep)

	app.editor.input.SetValue("v")
	updated, _ := app.Update(keyMsg("enter"))
	_ = updated.(App)

	require.Len(t, f.Vars, 2)
	assert.Equal(t, "NEW_KEY", f.Vars[1].Key)
}

func TestEditWithNoVarSelected(t *testing.T) {
	f := makeTestFile(".env")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("e"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestDeleteConfirmYes(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmDelete, app.mode)

	updated, _ = app.Update(keyMsg("y"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "one var should be deleted")
	assert.True(t, f.Modified)
}

func TestDeleteConfirmNo(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeConfirmDelete, app.mode)

	updated, _ = app.Update(keyMsg("n"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "no vars should be deleted")
}

func TestDeleteConfirmEscape(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("d"))
	app = updated.(App)

	updated, _ = app.Update(keyMsg("esc"))
	app = updated.(App)

	assert.Equal(t, ModeNormal, app.mode)
	assert.Len(t, f.Vars, 2, "no vars should be deleted on escape")
}

func TestHelpToggle(t *testing.T) {
	app := newTestApp(nil)

	updated, _ := app.Update(keyMsg("?"))
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	updated, _ = app.Update(keyMsg("esc"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestHelpToggleWithQuestionMark(t *testing.T) {
	app := newTestApp(nil)

	updated, _ := app.Update(keyMsg("?"))
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	updated, _ = app.Update(keyMsg("?"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestHelpExitWithQuit(t *testing.T) {
	app := newTestApp(nil)

	updated, _ := app.Update(keyMsg("?"))
	app = updated.(App)
	assert.Equal(t, ModeHelp, app.mode)

	updated, _ = app.Update(keyMsg("q"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}

func TestSearchFlowEscape(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("/"))
	app = updated.(App)
	assert.Equal(t, ModeSearching, app.mode)

	updated, _ = app.Update(keyMsg("esc"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "", app.varList.SearchQuery, "search should be cleared on escape")
}

func TestSearchFlowEnter(t *testing.T) {
	f := makeTestFile(".env", "FOO", "BAR", "BAZ")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("/"))
	app = updated.(App)
	assert.Equal(t, ModeSearching, app.mode)

	app.searchInput.SetValue("FOO")
	app.varList.SetSearch("FOO")

	updated, _ = app.Update(keyMsg("enter"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "FOO", app.varList.SearchQuery, "search should be preserved on enter")
}

func TestFocusSwitching(t *testing.T) {
	f := makeTestFile(".env", "FOO")
	app := newTestApp([]*model.EnvFile{f})

	assert.Equal(t, FocusFiles, app.focus)

	updated, _ := app.Update(keyMsg("l"))
	app = updated.(App)
	assert.Equal(t, FocusVars, app.focus)
	assert.True(t, app.varList.Focused)
	assert.False(t, app.fileList.Focused)

	updated, _ = app.Update(keyMsg("h"))
	app = updated.(App)
	assert.Equal(t, FocusFiles, app.focus)
	assert.True(t, app.fileList.Focused)
	assert.False(t, app.varList.Focused)
}

func TestToggleSecrets(t *testing.T) {
	f := makeTestFile(".env", "SECRET_KEY")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	assert.False(t, app.varList.ShowSecrets)

	updated, _ := app.Update(keyMsg("ctrl+s"))
	app = updated.(App)
	assert.True(t, app.varList.ShowSecrets)

	updated, _ = app.Update(keyMsg("ctrl+s"))
	app = updated.(App)
	assert.False(t, app.varList.ShowSecrets)
}

func TestToggleSecretsRefiltersKeptSearch(t *testing.T) {
	f := makeTestFile(".env", "API_TOKEN", "HUNTER_NAME")
	f.Vars[0].Value, f.Vars[0].IsSecret = "hunter2", true
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	for _, k := range []string{"/", "h", "u", "n", "t", "e", "r", "enter"} {
		updated, _ := app.Update(keyMsg(k))
		app = updated.(App)
	}
	require.Equal(t, "hunter", app.varList.SearchQuery)
	assert.Equal(t, 1, len(app.varList.displayItems), "the masked secret's value must not match")

	updated, _ := app.Update(keyMsg("ctrl+s"))
	app = updated.(App)
	assert.Equal(t, 2, len(app.varList.displayItems), "revealing secrets must re-filter the kept search")

	updated, _ = app.Update(keyMsg("ctrl+s"))
	app = updated.(App)
	assert.Equal(t, 1, len(app.varList.displayItems), "hiding secrets must re-filter the kept search")
}

func TestToggleSort(t *testing.T) {
	f := makeTestFile(".env", "ZZZ", "AAA", "MMM")
	app := newTestApp([]*model.EnvFile{f})
	app.focus = FocusVars

	assert.False(t, app.varList.SortAlpha)

	updated, _ := app.Update(keyMsg("o"))
	app = updated.(App)
	assert.True(t, app.varList.SortAlpha)

	updated, _ = app.Update(keyMsg("o"))
	app = updated.(App)
	assert.False(t, app.varList.SortAlpha)
}

func TestDeleteWithNoFile(t *testing.T) {
	app := newTestApp(nil)
	app.focus = FocusVars

	updated, _ := app.Update(keyMsg("d"))
	app = updated.(App)
	assert.Equal(t, ModeNormal, app.mode)
}
