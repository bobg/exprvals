package exprvals

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// Scan scans the given AST expression node to determine the values it can represent.
func Scan(node ast.Expr, files []*ast.File, info *types.Info) ([]constant.Value, bool) {
	node = ast.Unparen(node)

	tv, ok := info.Types[node]
	if ok && tv.IsValue() {
		return []constant.Value{tv.Value}, true
	}

	switch node := node.(type) {
	case *ast.Ident:
		return scanIdent(node, files, info)
	}

	return nil, false
}

func scanIdent(ident *ast.Ident, files []*ast.File, info *types.Info) ([]constant.Value, bool) {
	obj := info.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}

	switch obj := obj.(type) {
	case *types.Const:
		return []constant.Value{obj.Val()}, true

	case *types.Var:
		return scanVar(ident, obj, files, info)
	}

	return nil, false
}

// scanVar inspects the code leading up to the appearance of ident, which is a variable,
// to determine the possible constant values it can have.
// A boolean result of true means that all possible values were determined.
// A false result means that there may be some undetermined values.
// For example, in this code:
//
//	x := "hello"
//	if condition() {
//	  x = "goodbye"
//	}
//	return x
//
// scanVar can determine that, by the time the return statement is reached,
// x can be only "hello" or "goodbye" and nothing else.
// For comparison, in this code:
//
//	x := "hello"
//	doSomething(&x)
//	return x
//
// scanVar can determine that one possible value for x is "hello"
// but cannot determine that it is the only possible value.
func scanVar(ident *ast.Ident, v *types.Var, files []*ast.File, info *types.Info) ([]constant.Value, bool) {
}
