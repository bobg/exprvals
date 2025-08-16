package exprvals

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// Scan scans the given AST expression node to determine the values it can represent.
func Scan(node ast.Expr, info *types.Info) ([]constant.Value, bool) {
	node = ast.Unparen(node)

	tv, ok := info.Types[node]
	if ok && tv.IsValue() {
		return []constant.Value{tv.Value}, true
	}

	switch node := node.(type) {
	case *ast.Ident:
		return scanIdent(node, info)
	}

	return nil, false
}

func scanIdent(ident *ast.Ident, info *types.Info) ([]constant.Value, bool) {
	obj := info.ObjectOf(ident)
	if obj == nil { 
		return nil, false
	}

	switch obj := obj.(type) {
	case *types.Const:
		return []constant.Value{obj.Val()}, true

	case *types.Var:
		return scanVar(obj, info)
	}
}
