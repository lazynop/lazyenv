package tui

import (
	"testing"

	"github.com/lazynop/lazyenv/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestEditorStartEdit(t *testing.T) {
	e := NewEditorModel()
	v := &model.EnvVar{Key: "DB_HOST", Value: "localhost"}
	e.StartEdit(v, 3)

	assert.Equal(t, 3, e.varIndex)
	assert.False(t, e.isAdd)
	assert.Equal(t, "Edit DB_HOST: ", e.label)
	assert.Equal(t, "localhost", e.input.Value())
}

func TestEditorStartAdd(t *testing.T) {
	e := NewEditorModel()
	e.StartAdd()

	assert.True(t, e.isAdd)
	assert.Equal(t, addStepKey, e.addStep)
	assert.Equal(t, "", e.addKey)
	assert.Equal(t, "New key: ", e.label)
	assert.Equal(t, "", e.input.Value())
}

func TestEditorStartAddValue(t *testing.T) {
	e := NewEditorModel()
	e.StartAdd()
	e.StartAddValue("MY_KEY")

	assert.Equal(t, "MY_KEY", e.addKey)
	assert.Equal(t, addStepValue, e.addStep)
	assert.Equal(t, "MY_KEY = ", e.label)
	assert.Equal(t, "", e.input.Value())
}

func TestEditorFinishEdit(t *testing.T) {
	e := NewEditorModel()
	v := &model.EnvVar{Key: "FOO", Value: "old"}
	e.StartEdit(v, 2)
	e.input.SetValue("new")

	result := e.Finish()
	assert.Equal(t, "new", result.Value)
	assert.Equal(t, 2, result.VarIndex)
	assert.False(t, result.IsAdd)
	assert.False(t, result.Cancelled)
}

func TestEditorFinishAddKeyStep(t *testing.T) {
	e := NewEditorModel()
	e.StartAdd()
	e.input.SetValue("NEW_KEY")

	result := e.Finish()
	assert.True(t, result.IsAdd)
	assert.Equal(t, addStepKey, result.AddStep)
	assert.Equal(t, "NEW_KEY", result.Value)
}

func TestEditorFinishAddValueStep(t *testing.T) {
	e := NewEditorModel()
	e.StartAdd()
	e.StartAddValue("NEW_KEY")
	e.input.SetValue("some_value")

	result := e.Finish()
	assert.True(t, result.IsAdd)
	assert.Equal(t, addStepValue, result.AddStep)
	assert.Equal(t, "some_value", result.Value)
}

func TestEditorView(t *testing.T) {
	e := NewEditorModel()
	v := &model.EnvVar{Key: "FOO", Value: "bar"}
	e.StartEdit(v, 0)

	view := e.View()
	assert.Contains(t, view, "Edit FOO: ")
}
