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
	internalParserPath = "github.com/lazynop/lazyenv/internal/parser"
	internalUtilPath   = "github.com/lazynop/lazyenv/internal/util"
	bubbleteaPath      = "charm.land/bubbletea/v2"
)

// sourceRef is a reference, in this package's non-test source, to a function
// of an imported package or of this one.
type sourceRef struct {
	file  string // file name, relative to the package directory
	line  int
	owner string // enclosing top-level function, "" at package level
	name  string // "pkg.Func" (pkg being the last element of the import path), or "Func"
}

func (r sourceRef) String() string {
	return fmt.Sprintf("%s:%d: %s in %s", r.file, r.line, r.name, r.owner)
}

// sourceRefs returns every reference, in this package's non-test .go files, to
// one of funcs of the package at importPath, whatever name each file imports it
// under; with importPath "", every call to one of this package's own funcs. It
// walks the AST, so comments and strings never match.
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
		local := ""
		if importPath != "" {
			if local = importName(file, importPath); local == "" {
				continue
			}
		}
		for _, decl := range file.Decls {
			owner := ""
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = fn.Name.Name
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				fn := refIdent(n, local)
				if fn == nil || !slices.Contains(funcs, fn.Name) {
					return true
				}
				qualified := fn.Name
				if importPath != "" {
					qualified = path.Base(importPath) + "." + fn.Name
				}
				refs = append(refs, sourceRef{
					file:  name,
					line:  fset.Position(fn.Pos()).Line,
					owner: owner,
					name:  qualified,
				})
				return true
			})
		}
	}
	return refs
}

// refIdent returns the function name n refers to: the Func of a pkg.Func
// selector when local is the file's name for the imported package, or the
// callee of a Func(...) call to this package when local is "".
func refIdent(n ast.Node, local string) *ast.Ident {
	switch n := n.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := n.X.(*ast.Ident); ok && local != "" && pkg.Name == local {
			return n.Sel
		}
	case *ast.CallExpr:
		if id, ok := n.Fun.(*ast.Ident); ok && local == "" {
			return id
		}
	}
	return nil
}

// isSelector reports whether expr is pkg.name.
func isSelector(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
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

// assertRefsAllowed fails for every ref that no allowed entry covers, and for
// every entry that covers no ref, so a stale entry cannot linger. Keys are a
// file name ("app_disk.go") or "file:function" ("scanner.go:ScanDir"); values
// say why the entry is allowed. fix tells the reader what to do instead.
func assertRefsAllowed(t *testing.T, refs []sourceRef, allowed map[string]string, fix string) {
	t.Helper()
	used := make(map[string]bool, len(allowed))
	var violations []string
	for _, ref := range refs {
		key := ref.file
		if _, ok := allowed[key]; !ok {
			key = ref.file + ":" + ref.owner
		}
		if _, ok := allowed[key]; ok {
			used[key] = true
			continue
		}
		violations = append(violations, ref.String())
	}
	assert.Empty(t, violations, fix)
	for key := range allowed {
		assert.True(t, used[key], "allowed entry %q matches no reference: remove it", key)
	}
}
