package exprvals

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// Scan scans the given AST expression node to determine the values it can represent.
func Scan(node ast.Expr, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
	node = ast.Unparen(node)

	tv, ok := info.Types[node]
	if ok && tv.IsValue() {
		v := tv.Value
		if v == nil {
			return nil, false
		}
		return map[string]constant.Value{v.ExactString(): v}, true
	}

	switch node := node.(type) {
	case *ast.Ident:
		return scanIdent(node, files, info)
	}

	return nil, false
}

func scanIdent(ident *ast.Ident, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
	obj := info.ObjectOf(ident)
	if obj == nil {
		return nil, false
	}

	switch obj := obj.(type) {
	case *types.Const:
		v := obj.Val()
		return map[string]constant.Value{v.ExactString(): v}, true

	case *types.Var:
		return scanVar(ident, obj, files, info)
	}

	return nil, false
}

// scanVar inspects the code in the scope of ident, which is a variable,
// to determine the possible constant values it can have.
func scanVar(ident *ast.Ident, v *types.Var, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
	v = v.Origin()

	scope := v.Parent()

	// Find the smallest AST node containing all of scope.

	var node ast.Node
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			if !nodeContains(n, scope) {
				return false
			}
			if node == nil {
				node = n
				return true
			}
			if nodeSmaller(n, node) {
				node = n
			}
			return true
		})
	}

	if node == nil {
		return nil, false
	}

	// Find all assignments to v within node.
	var (
		vals     = make(map[string]constant.Value)
		complete = true
	)

	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		switch n := n.(type) {
		case *ast.AssignStmt:
			switch n.Tok {
			case token.ASSIGN, token.DEFINE:
				// Is v on the left-hand side?
				found := -1
				for i, lhs := range n.Lhs {
					if exprIsVar(lhs, v, info) {
						found = i
						break
					}
				}
				if found < 0 {
					return true
				}
				if len(n.Rhs) != len(n.Lhs) {
					// TODO: try to analyze the right-hand side anyway
					complete = false
					return true
				}
				rhsVals, ok := Scan(n.Rhs[found], files, info)
				for _, val := range rhsVals {
					vals[val.ExactString()] = val
				}
				complete = complete && ok

			default:
				// TODO: other assignment operators
				complete = false
			}

		case *ast.UnaryExpr:
			if n.Op != token.AND {
				return true
			}
			if !exprIsVar(n.X, v, info) {
				return true
			}
			complete = false
			// TODO: try to analyze what is done with the address of v

		case *ast.ValueSpec:
			// Is v on the left-hand side?
			found := -1
			for i, lhs := range n.Names {
				if identIsVar(lhs, v, info) {
					found = i
					break
				}
			}
			if found < 0 {
				return true
			}
			if len(n.Names) != len(n.Values) {
				// TODO: try to analyze the right-hand side anyway
				complete = false
				return true
			}
			rhsVals, ok := Scan(n.Values[found], files, info)
			for _, val := range rhsVals {
				vals[val.ExactString()] = val
			}
			complete = complete && ok

			// TODO: return statements too? (in case v is a result parameter, named or unnamed)
		}

		return true
	})

	return vals, complete
}

func exprIsVar(expr ast.Expr, v *types.Var, info *types.Info) bool {
	expr = ast.Unparen(expr)
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return identIsVar(id, v, info)
}

func identIsVar(id *ast.Ident, v *types.Var, info *types.Info) bool {
	obj := info.ObjectOf(id)
	if obj == nil {
		return false
	}
	vv, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	return vv.Origin() == v.Origin()
}

// Does a contain b?
func nodeContains(a, b ast.Node) bool {
	return a.Pos() <= b.Pos() && a.End() >= b.End()
}

// Does a cover a smaller extent than b?
func nodeSmaller(a, b ast.Node) bool {
	return a.Pos() > b.Pos() || a.End() < b.End()
}
