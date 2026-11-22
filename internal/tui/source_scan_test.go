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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	internalModelPath  = "github.com/lazynop/lazyenv/internal/model"
	internalParserPath = "github.com/lazynop/lazyenv/internal/parser"
	internalUtilPath   = "github.com/lazynop/lazyenv/internal/util"
	bubbleteaPath      = "charm.land/bubbletea/v2"
	thisPackage        = ""
)

type identMatcher func(n ast.Node, importedAs string) *ast.Ident

type (
	allowKey  string
	reason    string
	allowList map[allowKey]reason
)

func inFile(file string) allowKey { return allowKey(file) }

func inFunc(file, function string) allowKey { return allowKey(file + ":" + function) }

type sourceRef struct {
	file          string
	line          int
	enclosingFunc string
	name          string
}

func (r sourceRef) String() string {
	return fmt.Sprintf("%s:%d: %s in %s", r.file, r.line, r.name, r.enclosingFunc)
}

func sourceRefs(t *testing.T, importPath string, funcs ...string) []sourceRef {
	t.Helper()
	return sourceMatches(t, importPath, func(n ast.Node, importedAs string) *ast.Ident {
		if fn := refIdent(n, importedAs); fn != nil && slices.Contains(funcs, fn.Name) {
			return fn
		}
		return nil
	})
}

func sourceMatches(t *testing.T, importPath string, match identMatcher) []sourceRef {
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
		importedAs := thisPackage
		if importPath != thisPackage {
			if importedAs = importName(file, importPath); importedAs == "" {
				continue
			}
		}
		for _, decl := range file.Decls {
			enclosingFunc := ""
			if fn, ok := decl.(*ast.FuncDecl); ok {
				enclosingFunc = fn.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				id := match(n, importedAs)
				if id == nil {
					return true
				}
				qualified := id.Name
				if importPath != thisPackage {
					qualified = path.Base(importPath) + "." + id.Name
				}
				refs = append(refs, sourceRef{
					file:          name,
					line:          fset.Position(id.Pos()).Line,
					enclosingFunc: enclosingFunc,
					name:          qualified,
				})
				return true
			})
		}
	}
	return refs
}

func refIdent(n ast.Node, importedAs string) *ast.Ident {
	switch n := n.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := n.X.(*ast.Ident); ok && importedAs != thisPackage && pkg.Name == importedAs {
			return n.Sel
		}
	case *ast.CallExpr:
		if id, ok := n.Fun.(*ast.Ident); ok && importedAs == thisPackage {
			return id
		}
	}
	return nil
}

func isSelector(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

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

func assertRefsMatchAllowList(t *testing.T, refs []sourceRef, allowed allowList, whatToDoInstead string) {
	t.Helper()
	used := make(map[allowKey]bool, len(allowed))
	var violations []string
	for _, ref := range refs {
		key := inFile(ref.file)
		if _, ok := allowed[key]; !ok {
			key = inFunc(ref.file, ref.enclosingFunc)
		}
		if _, ok := allowed[key]; ok {
			used[key] = true
			continue
		}
		violations = append(violations, ref.String())
	}
	assert.Empty(t, violations, whatToDoInstead)
	for key := range allowed {
		assert.True(t, used[key], "allowed entry %q matches no reference: remove it", key)
	}
}
