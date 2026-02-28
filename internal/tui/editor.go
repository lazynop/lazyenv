package tui

import (
	"fmt"
	"gitlab.com/traveltoaiur/lazyenv/internal/model"

	"charm.land/bubbles/v2/textinput"
)

// Add step tracks the two-step add flow.
type addStep int

const (
	addStepKey   addStep = iota // entering key name
	addStepValue                // entering value
)

// EditorResult is returned when editing finishes.
type EditorResult struct {
	Value     string
	VarIndex  int
	IsAdd     bool
	AddStep   addStep
	Cancelled bool
}

// EditorModel manages inline editing of variable values.
type EditorModel struct {
	input    textinput.Model
	varIndex int
	isAdd    bool
	addStep  addStep
	addKey   string // stored key name during add flow
	label    string
}

// NewEditorModel creates a new editor model.
func NewEditorModel() EditorModel {
	ti := textinput.New()
	ti.CharLimit = 1000
	return EditorModel{
		input:    ti,
		varIndex: -1,
	}
}

// StartEdit begins editing an existing variable's value.
func (m *EditorModel) StartEdit(v *model.EnvVar, idx int) {
	m.input.SetValue(v.Value)
	m.input.CursorEnd()
	m.varIndex = idx
	m.isAdd = false
	m.label = fmt.Sprintf("Edit %s: ", v.Key)
	m.input.Placeholder = "new value"
}

// StartAdd begins the add flow (key name first).
func (m *EditorModel) StartAdd() {
	m.input.SetValue("")
	m.isAdd = true
	m.addStep = addStepKey
	m.addKey = ""
	m.label = "New key: "
	m.input.Placeholder = "KEY_NAME"
}

// StartAddValue moves to the value step of the add flow.
func (m *EditorModel) StartAddValue(key string) {
	m.addKey = key
	m.addStep = addStepValue
	m.input.SetValue("")
	m.label = fmt.Sprintf("%s = ", key)
	m.input.Placeholder = "value"
}

// Finish completes the current editing operation.
func (m *EditorModel) Finish() EditorResult {
	return EditorResult{
		Value:    m.input.Value(),
		VarIndex: m.varIndex,
		IsAdd:    m.isAdd,
		AddStep:  m.addStep,
	}
}

// View renders the editor.
func (m *EditorModel) View() string {
	return m.label + m.input.View()
}
