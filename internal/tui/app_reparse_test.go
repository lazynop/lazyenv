package tui

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
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
	reparseCarriesOver                           // runtime-only, copied from the old file
)

// envFileFieldFates classifies every model.EnvFile field. A new field fails
// TestReparseClassifiesEveryEnvFileField until it is listed here; a field
// marked reparseCarriesOver must also be copied in reparse (app_reparse.go).
var envFileFieldFates = map[string]reparseFieldFate{
	"Path":            reparseReadsFromDisk,
	"Name":            reparseReadsFromDisk,
	"Vars":            reparseReadsFromDisk,
	"Lines":           reparseReadsFromDisk,
	"TrailingNewline": reparseReadsFromDisk,
	"DeletedVars":     reparseDropsEdit,
	"Modified":        reparseDropsEdit,
	"GitWarning":      reparseCarriesOver,
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
	old, err := parser.ParseFile(path, config.SecretsConfig{})
	require.NoError(t, err)

	// Runtime-only state: every reparseCarriesOver field needs a non-zero value.
	old.GitWarning = true
	// Unsaved edits, which the re-parse discards.
	old.UpdateVar(0, "edited")
	old.DeleteVar(1)

	refreshed, err := reparse(old, config.SecretsConfig{})
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
	paths, err := filepath.Glob("*.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	var violations []string
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := goparser.ParseFile(fset, path, nil, 0)
		require.NoError(t, err)
		violations = append(violations, parseFileViolations(fset, path, file)...)
	}

	assert.Empty(t, violations, "re-parse an in-memory file with reparse (app_reparse.go) so its "+
		"runtime state survives; a load with no in-memory predecessor goes in parseFileAllowed")
}

// parseFileViolations returns "file:line" entries for every parser.ParseFile
// reference in file that sits outside an allowlisted function.
func parseFileViolations(fset *token.FileSet, path string, file *ast.File) []string {
	local := internalParserImportName(file)
	if local == "" {
		return nil
	}
	var out []string
	for _, decl := range file.Decls {
		owner := ""
		if fn, ok := decl.(*ast.FuncDecl); ok {
			owner = fn.Name.Name
		}
		if _, ok := parseFileAllowed[path+":"+owner]; ok {
			continue
		}
		ast.Inspect(decl, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "ParseFile" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == local {
				pos := fset.Position(sel.Pos())
				out = append(out, fmt.Sprintf("%s:%d: parser.ParseFile in %s", pos.Filename, pos.Line, owner))
			}
			return true
		})
	}
	return out
}

// internalParserImportName returns the name file uses for the internal parser
// package, or "" when file does not import it.
func internalParserImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p != "github.com/lazynop/lazyenv/internal/parser" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "parser"
	}
	return ""
}
