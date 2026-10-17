package tui

import (
	"reflect"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/stretchr/testify/assert"
)

func TestKeyLabel(t *testing.T) {
	for k, want := range map[string]string{
		"q":      "q",
		"?":      "?",
		"ctrl+c": "Ctrl+C",
		"up":     "↑",
		"pgdown": "PgDn",
		"enter":  "Enter",
	} {
		assert.Equal(t, want, KeyLabel(k), k)
	}
}

func TestEveryBoundKeyHasALabel(t *testing.T) {
	keyMap := reflect.ValueOf(DefaultKeyMap())
	for field := range keyMap.Type().Fields() {
		for _, k := range keyMap.FieldByIndex(field.Index).Interface().(key.Binding).Keys() {
			_, ok := keyLabel(k)
			assert.True(t, ok, "KeyMap.%s binds %q, which has no label: add it to keyLabels", field.Name, k)
		}
	}
}
