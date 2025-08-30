// Package exprvals provides a way to scan Go AST expressions for the values they can represent.
package exprvals

import (
	"go/ast"
	"go/constant"

	"golang.org/x/tools/go/packages"
)

type Value interface {
	ExactString() string
}

type Pointer struct {
	Elem Value
}

func (v Pointer) ExactString() string {
	return "&" + v.Elem.ExactString()
}

type Map = map[string]Value

// Scan scans the given AST expression node to determine the values it might represent.
// If the node is a boolean, string, or number literal, that is its value.
// If it is an identifier that refers to a constant, Scan returns that value.
// If it is an identifier that refers to a variable,
// Scan looks at all the assignments to that variable to determine the possible values.
// In the future, other types of expression may be supported.
//
// The result is a Map.
// Each key is the string representation (using ExactString) of the value.
// This function also returns a boolean indicating whether all possible values were determined.
//
// For example, given the following code:
//
//	x := "hello"
//	if condition() {
//	  x = "goodbye"
//	}
//	return x
//
// Scan can determine that, by the time the return statement is reached,
// x can be only "hello" or "goodbye" and nothing else.
func Scan(node ast.Expr, pkg *packages.Package) (Map, bool) {
	node = ast.Unparen(node)

	tv, ok := pkg.TypesInfo.Types[node]
	if ok && tv.IsValue() {
		if v := tv.Value; v != nil && v.Kind() != constant.Unknown {
			return map[string]Value{v.ExactString(): v}, true
		}
	}

	switch node := node.(type) {
	case *ast.CallExpr:
		if !isSingleValue(node, pkg) {
			return nil, false
		}
		return ScanCallResult(node, 0, pkg)

	case *ast.Ident:
		return scanIdent(node, pkg)

	case *ast.UnaryExpr:
		return scanUnaryExpr(node, pkg)

	case *ast.BinaryExpr:
		return scanBinaryExpr(node, pkg)
	}

	return nil, false
}
