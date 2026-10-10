package tui

import (
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const internalParserPath = "github.com/lazynop/lazyenv/internal/parser"

// sourceRef is a reference, in this package's non-test source, to a function
// of an imported package.
type sourceRef struct {
	file  string // file name, relative to the package directory
	line  int
	owner string // enclosing top-level function, "" at package level
	name  string // "pkg.Func", pkg being the last element of the import path
}

func (r sourceRef) String() string {
	return fmt.Sprintf("%s:%d: %s in %s", r.file, r.line, r.name, r.owner)
}

// sourceRefs returns every reference to one of funcs of the package at
// importPath in this package's non-test .go files, whatever name each file
// imports it under. It walks the AST, so comments and strings never match.
func sourceRefs(t *testing.T, importPath string, funcs ...string) []sourceRef {
	t.Helper()
	names, err := filepath.Glob("*.go")
	require.NoError(t, err)

	fset := token.NewFileSet()
	var refs []sourceRef
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := goparser.ParseFile(fset, name, nil, goparser.SkipObjectResolution)
		require.NoError(t, err)
		local := importName(file, importPath)
		if local == "" {
			continue
		}
		for _, decl := range file.Decls {
			owner := ""
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = fn.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || !slices.Contains(funcs, sel.Sel.Name) {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == local {
					refs = append(refs, sourceRef{
						file:  name,
						line:  fset.Position(sel.Pos()).Line,
						owner: owner,
						name:  path.Base(importPath) + "." + sel.Sel.Name,
					})
				}
				return true
			})
		}
	}
	return refs
}

// importName returns the name file uses for the package at importPath, or ""
// when file does not import it.
func importName(file *ast.File, importPath string) string {
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p != importPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return path.Base(importPath)
	}
	return ""
}
