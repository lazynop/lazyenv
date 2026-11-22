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

type reparseFieldFate int

const (
	reparseReadsFromDisk reparseFieldFate = iota
	reparseDropsEdit
	reparseCarriesOver
)

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

var parseFileAllowed = allowList{
	inFunc("app_reparse.go", "reparse"):           "the helper itself",
	inFunc("scanner.go", "ScanDir"):               "initial load; CheckGitIgnore runs on the result",
	inFunc("app_fileinput.go", "finaliseNewFile"): "a newly created file; CheckGitIgnore runs on the result",
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
	old, err := parser.ParseFile(path, config.SecretsConfig{ExtraPatterns: []string{"_HOST"}})
	require.NoError(t, err)
	old.GitWarning = true
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

func TestParseFileOnlyThroughReparse(t *testing.T) {
	assertRefsMatchAllowList(t, sourceRefs(t, internalParserPath, "ParseFile"), parseFileAllowed,
		"re-parse an in-memory file with reparse (app_reparse.go) so its runtime state survives; "+
			"a load with no in-memory predecessor goes in parseFileAllowed")
}

func TestEnvFilesComeFromTheParser(t *testing.T) {
	assert.Empty(t, sourceMatches(t, internalModelPath, envFileBuild),
		"get an EnvFile from parser.ParseFile or parser.ParseBytes (reparse for a file already in memory), never build one")
}

func envFileBuild(n ast.Node, modelImportName string) *ast.Ident {
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
	if !isSelector(typ, modelImportName, "EnvFile") {
		return nil
	}
	return typ.(*ast.SelectorExpr).Sel
}
