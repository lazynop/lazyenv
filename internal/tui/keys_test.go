package tui

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	for _, b := range keyMapBindings(t) {
		for _, k := range b.binding.Keys() {
			_, ok := keyLabel(k)
			assert.True(t, ok, "KeyMap.%s binds %q, which has no label: add it to keyLabels", b.field, k)
		}
	}
}

// TestEveryBoundKeyCanBePressed checks that keyMsg, which the tests press keys
// with, builds for every key a KeyMap binding uses a press that reads as that
// key, as key.Matches compares it.
func TestEveryBoundKeyCanBePressed(t *testing.T) {
	for _, b := range keyMapBindings(t) {
		for _, k := range b.binding.Keys() {
			var msg tea.KeyPressMsg
			if !assert.NotPanics(t, func() { msg = keyMsg(k) }, "KeyMap.%s binds %q, which keyMsg cannot press", b.field, k) {
				continue
			}
			assert.Equal(t, k, msg.String(), "keyMsg(%q) builds a different key, which KeyMap.%s does not match", k, b.field)
		}
	}
}

// TestKeyPressesCarryTheirCode fails on a tea.KeyPressMsg literal, in this
// package's test files, that sets Text but no Code: no terminal sends one, and
// a named key pressed that way ({Text: "enter"}) still matches its binding
// through String() while a handler reading Code sees no key at all. It checks
// elided literals too, as in []tea.KeyPressMsg{{Text: "q"}}.
func TestKeyPressesCarryTheirCode(t *testing.T) {
	names, err := filepath.Glob("*_test.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	var textOnly []string
	for _, name := range names {
		file, err := goparser.ParseFile(fset, name, nil, goparser.SkipObjectResolution)
		require.NoError(t, err)
		local := importName(file, bubbleteaPath)
		if local == "" {
			continue
		}
		isKeyPress := func(expr ast.Expr) bool {
			if star, ok := expr.(*ast.StarExpr); ok {
				expr = star.X
			}
			return isSelector(expr, local, "KeyPressMsg")
		}
		check := func(lit *ast.CompositeLit) {
			if fields := keyedFields(lit); fields["Text"] && !fields["Code"] {
				textOnly = append(textOnly, fset.Position(lit.Pos()).String())
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if isKeyPress(lit.Type) {
				check(lit)
			}
			if isKeyPress(elemType(lit.Type)) {
				for _, elt := range lit.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						elt = kv.Value
					}
					if inner, ok := elt.(*ast.CompositeLit); ok && inner.Type == nil {
						check(inner)
					}
				}
			}
			return true
		})
	}
	assert.Empty(t, textOnly, "press keys with keyMsg(k), or press(app, k...), which build the press a terminal sends")
}

// elemType returns the element type of a slice, array or map type, or nil.
func elemType(typ ast.Expr) ast.Expr {
	switch typ := typ.(type) {
	case *ast.ArrayType:
		return typ.Elt
	case *ast.MapType:
		return typ.Value
	}
	return nil
}

// keyedFields returns the field names a struct literal sets by key.
func keyedFields(lit *ast.CompositeLit) map[string]bool {
	fields := make(map[string]bool, len(lit.Elts))
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok {
				fields[id.Name] = true
			}
		}
	}
	return fields
}

// keyMapBinding is one field of the KeyMap: its name and its binding.
type keyMapBinding struct {
	field   string
	binding key.Binding
}

// keyMapBindings returns every field of DefaultKeyMap, in order, failing if
// one is unexported or not a key.Binding.
func keyMapBindings(t *testing.T) []keyMapBinding {
	t.Helper()
	var bindings []keyMapBinding
	keyMap := reflect.ValueOf(DefaultKeyMap())
	for field := range keyMap.Type().Fields() {
		require.True(t, field.IsExported(), "KeyMap.%s is unexported, so reflect's Interface() cannot take its value", field.Name)
		binding, ok := keyMap.FieldByIndex(field.Index).Interface().(key.Binding)
		require.True(t, ok, "KeyMap.%s is not a key.Binding", field.Name)
		bindings = append(bindings, keyMapBinding{field.Name, binding})
	}
	return bindings
}

// keyCodes are the codes of the key names longer than one character that
// keyMsg knows, other than Ctrl chords.
var keyCodes = map[string]rune{
	"enter": tea.KeyEnter, "esc": tea.KeyEscape, "tab": tea.KeyTab, "space": tea.KeySpace,
	"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight,
	"pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd,
	"f5": tea.KeyF5,
}

// keyMsg returns the key press a terminal delivers for the key a binding
// names: a special key carries its code (Space also its text), a Ctrl chord
// its letter and the modifier, an uppercase letter the lowercase code with
// Shift, any other single character its code and text. It panics on a name it
// does not know, so a misspelt key fails the test instead of being typed.
func keyMsg(k string) tea.KeyPressMsg {
	if c, ok := strings.CutPrefix(k, "ctrl+"); ok && len([]rune(c)) == 1 {
		return tea.KeyPressMsg{Code: []rune(c)[0], Mod: tea.ModCtrl}
	}
	if code, ok := keyCodes[k]; ok {
		if code == tea.KeySpace {
			return tea.KeyPressMsg{Code: code, Text: " "}
		}
		return tea.KeyPressMsg{Code: code}
	}
	if r := []rune(k); len(r) == 1 {
		if unicode.IsUpper(r[0]) {
			return tea.KeyPressMsg{Code: unicode.ToLower(r[0]), ShiftedCode: r[0], Mod: tea.ModShift, Text: k}
		}
		return tea.KeyPressMsg{Code: r[0], Text: k}
	}
	panic(fmt.Sprintf("keyMsg: cannot build %q: check its spelling; a new named key goes in keyCodes, a new modifier in keyMsg", k))
}
