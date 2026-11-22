package tui

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pointerMethodSet map[string]map[string]bool

func typeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func pointerReceiverMethods(t *testing.T) pointerMethodSet {
	t.Helper()
	methods := pointerMethodSet{}
	sourceMatches(t, thisPackage, func(n ast.Node, _ string) *ast.Ident {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return nil
		}
		if _, ok := fn.Recv.List[0].Type.(*ast.StarExpr); !ok {
			return nil
		}
		recv := typeName(fn.Recv.List[0].Type)
		if methods[recv] == nil {
			methods[recv] = map[string]bool{}
		}
		methods[recv][fn.Name.Name] = true
		return nil
	})
	return methods
}

func (s pointerMethodSet) has(recvType, method string) bool {
	if recvType != "" {
		return s[recvType][method]
	}
	for _, names := range s {
		if names[method] {
			return true
		}
	}
	return false
}

func declaredTypes(fn *ast.FuncDecl) map[string]string {
	types := map[string]string{}
	var fields []*ast.Field
	if fn.Recv != nil {
		fields = append(fields, fn.Recv.List...)
	}
	fields = append(fields, fn.Type.Params.List...)
	for _, field := range fields {
		for _, name := range field.Names {
			types[name.Name] = typeName(field.Type)
		}
	}
	return types
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

func readOfPointerCallReceiver(ret *ast.ReturnStmt, varTypes map[string]string, pointerMethods pointerMethodSet) *ast.Ident {
	called := map[string]bool{}
	notReads := map[*ast.Ident]bool{}
	inspectSkippingFuncLits(ret.Results, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			notReads[sel.Sel] = true
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		recv, ok := sel.X.(*ast.Ident)
		if !ok || !pointerMethods.has(varTypes[recv.Name], sel.Sel.Name) {
			return true
		}
		called[recv.Name] = true
		notReads[recv] = true
		inspectSkippingFuncLits(call.Args, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				notReads[id] = true
			}
			return true
		})
		return true
	})
	var read *ast.Ident
	inspectSkippingFuncLits(ret.Results, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && called[id.Name] && !notReads[id] {
			read = id
		}
		return read == nil
	})
	return read
}

func TestReturnsCallPointerMethodsBeforeReadingTheirReceiver(t *testing.T) {
	pointerMethods := pointerReceiverMethods(t)
	var varTypes map[string]string
	refs := sourceMatches(t, thisPackage, func(n ast.Node, _ string) *ast.Ident {
		switch n := n.(type) {
		case *ast.FuncDecl:
			varTypes = declaredTypes(n)
		case *ast.ReturnStmt:
			return readOfPointerCallReceiver(n, varTypes, pointerMethods)
		}
		return nil
	})
	var violations []string
	for _, ref := range refs {
		violations = append(violations, ref.String())
	}
	assert.Empty(t, violations,
		"a return reads a variable and calls a pointer-receiver method on it: Go leaves their order unspecified, so call the method in a statement of its own first")
}
