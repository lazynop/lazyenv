package tui

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
)

// The property: in the main view the file list's cursor sits on the active
// file (FileListModel.Cursor == Selected) and the var list shows that file, so
// the highlighted file is the one the var list shows; selectCursorFile relies
// on it to reload the var list only on another file. Modes that move the
// cursor without changing the active file (the compare picker, the matrix)
// must put it back on the way out. Every pair of inputs from the main view is
// tried, then the keys that leave or answer a mode, checking the property
// after each input that lands in the main view. Bindings come from KeyMap by
// reflection, so a new one is covered.

// cursorInput is one input the property test sends: a key press or a wheel tick.
type cursorInput struct {
	name string
	msg  tea.Msg
}

// cursorInputs returns the first key of every KeyMap binding, plus a wheel
// tick down over each panel.
func cursorInputs(t *testing.T, app App) []cursorInput {
	t.Helper()
	var inputs []cursorInput
	for _, b := range keyMapBindings(t) {
		k := b.binding.Keys()[0]
		inputs = append(inputs, cursorInput{k, keyMsg(k)})
	}
	return append(inputs,
		cursorInput{"wheel over files", tea.MouseWheelMsg{X: 1, Y: 3, Button: tea.MouseWheelDown}},
		cursorInput{"wheel over vars", tea.MouseWheelMsg{X: app.fileWidth + 5, Y: 3, Button: tea.MouseWheelDown}},
	)
}

// cursorFollowUps answer or leave whatever mode a pair of inputs reached:
// each list is tried after every pair, the second one first going on with
// Enter (into the diff view from the compare picker).
var cursorFollowUps = [][]string{
	{"y", "esc", "esc", "q"},
	{"enter", "y", "esc", "esc", "q"},
}

func TestMainViewCursorStaysOnTheActiveFile(t *testing.T) {
	newApp := func(t *testing.T) App {
		return loadFixtureApp(t, config.DefaultConfig(), readOnlyFixture, 120, 40)
	}
	inputs := cursorInputs(t, newApp(t))
	for _, first := range inputs {
		for _, second := range inputs {
			for _, followUps := range cursorFollowUps {
				t.Run(fmt.Sprintf("%s then %s then %v", first.name, second.name, followUps), func(t *testing.T) {
					t.Parallel() // each case owns its temp dir and App
					app := newApp(t)
					sent := []cursorInput{first, second}
					for _, k := range followUps {
						sent = append(sent, cursorInput{k, keyMsg(k)})
					}
					for i, in := range sent {
						updated, _ := app.Update(in.msg)
						app = updated.(App)
						if app.mode == ModeNormal {
							require.Equal(t, app.fileList.Selected, app.fileList.Cursor,
								"after %v the cursor is off the active file", names(sent[:i+1]))
							require.Same(t, app.fileList.SelectedFile(), app.varList.File,
								"after %v the var list shows another file than the active one", names(sent[:i+1]))
						}
					}
				})
			}
		}
	}
}

func names(inputs []cursorInput) []string {
	out := make([]string, len(inputs))
	for i, in := range inputs {
		out[i] = in.name
	}
	return out
}
