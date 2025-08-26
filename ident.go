package exprvals

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func scanIdent(id *ast.Ident, pkg *packages.Package) (Map, bool) {
	obj := pkg.TypesInfo.ObjectOf(id)
	if obj == nil {
		return nil, false
	}
	v, ok := obj.(*types.Var)
	if !ok {
		return nil, false
	}
	return scanVarAt(v, id.Pos(), pkg)
}

func scanVarAt(v *types.Var, pos token.Pos, pkg *packages.Package) (Map, bool) {
	// xxx

	return nil, false
}
