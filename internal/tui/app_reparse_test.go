package tui

import (
	"go/ast"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lazynop/lazyenv/internal/config"
	"github.com/lazynop/lazyenv/internal/model"
	"github.com/lazynop/lazyenv/internal/parser"
)

// reparseFieldFate says what happens to a model.EnvFile field when reparse
// replaces an in-memory file with a fresh parse.
type reparseFieldFate int

const (
	reparseReadsFromDisk reparseFieldFate = iota // the parser fills it in
	reparseDropsEdit                             // unsaved edit state, cleared
	reparseCarriesOver                           // kept from the old file, which the disk does not hold
)

// envFileFieldFates classifies every model.EnvFile field. A new field fails
// TestReparseClassifiesEveryEnvFileField until it is listed here; a field
// marked reparseCarriesOver must also be kept by reparse (app_reparse.go).
var envFileFieldFates = map[string]reparseFieldFate{
	"Path":            reparseReadsFromDisk,
	"Name":            reparseReadsFromDisk,
	"Vars":            reparseReadsFromDisk,
	"Lines":           reparseReadsFromDisk,
	"TrailingNewline": reparseReadsFromDisk,
	"DeletedVars":     reparseDropsEdit,
	"Modified":        reparseDropsEdit,
	"GitWarning":      reparseCarriesOver,
	"Secrets":         reparseCarriesOver,
}

// parseFileAllowed lists the functions that may call parser.ParseFile
// directly, keyed by "file:function". Only reparse replaces an in-memory file;
// the others load a file that has no in-memory predecessor, so there is no
// runtime state to carry over.
var parseFileAllowed = map[string]string{
	"app_reparse.go:reparse":           "the helper itself",
	"scanner.go:ScanDir":               "initial load; CheckGitIgnore runs on the result",
	"app_fileinput.go:finaliseNewFile": "a newly created file; CheckGitIgnore runs on the result",
}

func TestReparseClassifiesEveryEnvFileField(t *testing.T) {
	typ := reflect.TypeFor[model.EnvFile]()
	fields := make(map[string]bool, typ.NumField())
	for f := range typ.Fields() {
		fields[f.Name] = true
		_, ok := envFileFieldFates[f.Name]
		assert.True(t, ok, "model.EnvFile.%s is not in envFileFieldFates: decide whether "+
			"the parser fills it, a re-parse drops it as edit state, or reparse "+
			"(app_reparse.go) must carry it over", f.Name)
	}
	for name := range envFileFieldFates {
		assert.True(t, fields[name], "envFileFieldFates lists %s, which model.EnvFile no longer has", name)
	}
}

func TestReparseKeepsRuntimeStateAndDropsEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("FOO=bar\nBAZ=qux\n"), 0o644))
	// Every reparseCarriesOver field needs a non-zero value: a custom [secrets]
	// config, and the runtime-only state.
	old, err := parser.ParseFile(path, config.SecretsConfig{ExtraPatterns: []string{"_HOST"}})
	require.NoError(t, err)
	old.GitWarning = true
	// Unsaved edits, which the re-parse discards.
	old.UpdateVar(0, "edited")
	old.DeleteVar(1)

	refreshed, err := reparse(old)
	require.NoError(t, err)

	oldV, newV := reflect.ValueOf(old).Elem(), reflect.ValueOf(refreshed).Elem()
	for name, fate := range envFileFieldFates {
		switch fate {
		case reparseCarriesOver:
			require.False(t, oldV.FieldByName(name).IsZero(),
				"give %s a non-zero value in this test's setup", name)
			assert.Equal(t, oldV.FieldByName(name).Interface(), newV.FieldByName(name).Interface(),
				"reparse must carry %s over from the in-memory file", name)
		case reparseDropsEdit:
			assert.True(t, newV.FieldByName(name).IsZero(), "reparse must drop the edit state in %s", name)
		}
	}

	require.Len(t, refreshed.Vars, 2)
	assert.Equal(t, "bar", refreshed.Vars[0].Value, "values come from disk, not memory")
	assert.Equal(t, "qux", refreshed.Vars[1].Value, "deleted vars come back from disk")
}

// TestParseFileOnlyThroughReparse fails on any parser.ParseFile reference in
// this package's non-test files outside parseFileAllowed: a direct re-parse
// returns a fresh EnvFile that silently loses runtime state like GitWarning.
func TestParseFileOnlyThroughReparse(t *testing.T) {
	assertRefsAllowed(t, sourceRefs(t, internalParserPath, "ParseFile"), parseFileAllowed,
		"re-parse an in-memory file with reparse (app_reparse.go) so its runtime state survives; "+
			"a load with no in-memory predecessor goes in parseFileAllowed")
}

// TestEnvFilesComeFromTheParser fails on a model.EnvFile built in this
// package's non-test files: by a composite literal, new or a var declaration.
// A file built by hand lacks what the parser sets up, such as each variable's
// flags, derived from its key and value with the [secrets] config.
func TestEnvFilesComeFromTheParser(t *testing.T) {
	assert.Empty(t, sourceMatches(t, internalModelPath, envFileBuild),
		"get an EnvFile from parser.ParseFile or parser.ParseBytes (reparse for a file already in memory), never build one")
}

// envFileBuild returns the EnvFile identifier of a model.EnvFile that n
// builds, local being the file's name for the model package, or nil.
func envFileBuild(n ast.Node, local string) *ast.Ident {
	var typ ast.Expr
	switch n := n.(type) {
	case *ast.CompositeLit:
		typ = n.Type
	case *ast.CallExpr:
		if fn, ok := n.Fun.(*ast.Ident); ok && fn.Name == "new" && len(n.Args) == 1 {
			typ = n.Args[0]
		}
	case *ast.ValueSpec:
		typ = n.Type
	}
	if !isSelector(typ, local, "EnvFile") {
		return nil
	}
	return typ.(*ast.SelectorExpr).Sel
}
