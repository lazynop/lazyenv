package tui

import (
	"go/ast"
	"go/build"
	"go/importer"
	goparser "go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type typedSource struct {
	fset  *token.FileSet
	files []*ast.File
	info  *types.Info
}

func typeCheckThisPackage(t *testing.T) typedSource {
	t.Helper()
	pkg, err := build.ImportDir(".", 0)
	require.NoError(t, err)

	src := typedSource{
		fset: token.NewFileSet(),
		info: &types.Info{
			Uses:       map[*ast.Ident]types.Object{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
		},
	}
	for _, name := range pkg.GoFiles {
		file, err := goparser.ParseFile(src.fset, name, nil, goparser.SkipObjectResolution)
		require.NoError(t, err)
		src.files = append(src.files, file)
	}
	conf := types.Config{Importer: importer.ForCompiler(src.fset, "source", nil)}
	_, err = conf.Check(pkg.ImportPath, src.fset, src.files, src.info)
	require.NoError(t, err)
	return src
}

func inspectSkippingFuncLits(exprs []ast.Expr, visit func(ast.Node) bool) {
	for _, expr := range exprs {
		ast.Inspect(expr, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok || n == nil {
				return false
			}
			return visit(n)
		})
	}
}

func fieldChainRoot(expr ast.Expr, info *types.Info) *ast.Ident {
	for {
		switch e := ast.Unparen(expr).(type) {
		case *ast.Ident:
			if _, ok := info.Uses[e].(*types.Var); ok {
				return e
			}
			return nil
		case *ast.SelectorExpr:
			if sel := info.Selections[e]; sel == nil || sel.Kind() != types.FieldVal {
				return nil
			}
			expr = e.X
		default:
			return nil
		}
	}
}

func pointerCallReceiver(call *ast.CallExpr, info *types.Info) *ast.Ident {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	selection := info.Selections[sel]
	if selection == nil || selection.Kind() != types.MethodVal {
		return nil
	}
	recv := selection.Obj().(*types.Func).Signature().Recv()
	if _, ok := recv.Type().(*types.Pointer); !ok {
		return nil
	}
	return fieldChainRoot(sel.X, info)
}

func readOfPointerCallReceiver(ret *ast.ReturnStmt, info *types.Info) *ast.Ident {
	callsOn := map[types.Object][]*ast.CallExpr{}
	receivers := map[*ast.Ident]bool{}
	inspectSkippingFuncLits(ret.Results, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if recv := pointerCallReceiver(call, info); recv != nil {
				callsOn[info.Uses[recv]] = append(callsOn[info.Uses[recv]], call)
				receivers[recv] = true
			}
		}
		return true
	})
	var read *ast.Ident
	inspectSkippingFuncLits(ret.Results, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok || receivers[id] {
			return true
		}
		for _, call := range callsOn[info.Uses[id]] {
			if id.Pos() < call.Pos() || id.End() > call.End() {
				read = id
			}
		}
		return read == nil
	})
	return read
}

func TestReturnsCallPointerMethodsBeforeReadingTheirReceiver(t *testing.T) {
	src := typeCheckThisPackage(t)
	var violations []string
	for _, file := range src.files {
		ast.Inspect(file, func(n ast.Node) bool {
			if ret, ok := n.(*ast.ReturnStmt); ok {
				if read := readOfPointerCallReceiver(ret, src.info); read != nil {
					violations = append(violations, src.fset.Position(read.Pos()).String())
				}
			}
			return true
		})
	}
	assert.Empty(t, violations,
		"a return reads a variable and calls a pointer-receiver method on it or on one of its fields: Go leaves their order unspecified, so call the method in a statement of its own first")
}
