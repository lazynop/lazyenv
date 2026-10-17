package tui

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"

	"github.com/lazynop/lazyenv/internal/model"
)

// Add step tracks the two-step add flow.
type addStep int

const (
	addStepKey   addStep = iota // entering key name
	addStepValue                // entering value
)

// EditorResult is returned when editing finishes.
type EditorResult struct {
	Value       string
	VarIndex    int
	IsAdd       bool
	IsRenameKey bool
	AddStep     addStep
	Cancelled   bool
	// Untouched is set for a value edit confirmed as the bar showed it. The
	// bar can't show every value as is (it turns line breaks into spaces and
	// drops control characters), so writing it back could change the value.
	Untouched bool
}

// EditorModel manages inline editing of variable values.
type EditorModel struct {
	input       textinput.Model
	varIndex    int
	isAdd       bool
	isRenameKey bool
	addStep     addStep
	addKey      string // stored key name during add flow
	label       string
	shown       string // the value as the bar showed it when the edit started
}

// NewEditorModel creates a new editor model.
func NewEditorModel() EditorModel {
	ti := textinput.New() // no CharLimit: a value of any length must load whole
	return EditorModel{
		input:    ti,
		varIndex: -1,
	}
}

// StartEdit begins editing an existing variable's value.
func (m *EditorModel) StartEdit(v *model.EnvVar, idx int) {
	m.input.SetValue(v.Value)
	m.shown = m.input.Value()
	m.input.CursorEnd()
	m.varIndex = idx
	m.isAdd = false
	m.isRenameKey = false
	m.label = fmt.Sprintf("Edit %s: ", v.Key)
	m.input.Placeholder = "new value"
}

// StartEditKey begins editing an existing variable's key name.
func (m *EditorModel) StartEditKey(v *model.EnvVar, idx int) {
	m.input.SetValue(v.Key)
	m.input.CursorEnd()
	m.varIndex = idx
	m.isAdd = false
	m.isRenameKey = true
	m.label = "Rename key: "
	m.input.Placeholder = "NEW_KEY_NAME"
}

// StartAdd begins the add flow (key name first).
func (m *EditorModel) StartAdd() {
	m.input.SetValue("")
	m.isAdd = true
	m.isRenameKey = false
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
		Value:       m.input.Value(),
		VarIndex:    m.varIndex,
		IsAdd:       m.isAdd,
		IsRenameKey: m.isRenameKey,
		AddStep:     m.addStep,
		Untouched:   !m.isAdd && !m.isRenameKey && m.input.Value() == m.shown,
	}
}
