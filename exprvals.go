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

// func getFuncForCall(call *ast.CallExpr, info *types.Info) *types.Func {
// 	var (
// 		f      = ast.Unparen(call.Fun)
// 		funObj types.Object
// 	)

// 	switch f := f.(type) {
// 	case *ast.Ident:
// 		funObj = info.ObjectOf(f)

// 	case *ast.SelectorExpr:
// 		if s, ok := info.Selections[f]; ok {
// 			funObj = s.Obj()
// 		}
// 	}
// 	if funObj == nil {
// 		return nil
// 	}

// 	fun, _ := funObj.(*types.Func)
// 	return fun
// }

// // ScanCallResult performs a [Scan] on the idx'th result of the given call expression.
// func ScanCallResult(call *ast.CallExpr, idx int, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
// 	// xxx bounds checking

// 	fun := getFuncForCall(call, info)
// 	if fun == nil {
// 		return nil, false
// 	}

// 	sig := fun.Signature()
// 	if sig == nil {
// 		return nil, false
// 	}

// 	scope := fun.Scope()
// 	if scope == nil {
// 		return nil, false
// 	}

// 	bodyNode := findSmallestEnclosingNode(files, scope)
// 	switch n := bodyNode.(type) {
// 	case *ast.FuncDecl:
// 		bodyNode = n.Body
// 	case *ast.FuncLit:
// 		bodyNode = n.Body
// 	}
// 	if bodyNode == nil {
// 		return nil, false
// 	}

// 	body, ok := bodyNode.(*ast.BlockStmt)
// 	if !ok {
// 		return nil, false
// 	}

// 	var (
// 		result   = make(map[string]constant.Value)
// 		complete = true
// 	)

// 	sigResults := sig.Results()
// 	if sigResults == nil {
// 		return nil, false
// 	}
// 	// xxx bounds checking
// 	nthResult := sigResults.At(idx)

// 	ast.Inspect(body, func(n ast.Node) bool {
// 		if n == nil {
// 			return false
// 		}
// 		switch n := n.(type) {
// 		case *ast.ReturnStmt:
// 			switch len(n.Results) {
// 			case 0:
// 				return true

// 			case 1:
// 				retExpr := ast.Unparen(n.Results[0])

// 				// Assign retExpr can produce as many values as sig wants to return.

// 				switch retExpr := retExpr.(type) {
// 				case *ast.CallExpr:
// 					vals, ok := ScanCallResult(retExpr, idx, files, info)
// 					for _, v := range vals {
// 						result[v.ExactString()] = v
// 					}
// 					complete = complete && ok

// 				default:
// 					vals, ok := Scan(retExpr, files, info)
// 					for _, v := range vals {
// 						result[v.ExactString()] = v
// 					}
// 					complete = complete && ok
// 				}

// 			default:
// 				// xxx bounds checking
// 				vals, ok := Scan(n.Results[idx], files, info)
// 				for _, v := range vals {
// 					result[v.ExactString()] = v
// 				}
// 				complete = complete && ok
// 			}

// 		case *ast.AssignStmt:
// 			vals, ok := scanAssignment(n, nthResult, files, info)
// 			for _, v := range vals {
// 				result[v.ExactString()] = v
// 			}
// 			complete = complete && ok
// 		}
// 		return true
// 	})

// 	return result, complete
// }

// func scanIdent(ident *ast.Ident, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
// 	obj := info.ObjectOf(ident)
// 	if obj == nil {
// 		return nil, false
// 	}

// 	switch obj := obj.(type) {
// 	case *types.Const:
// 		v := obj.Val()
// 		return map[string]constant.Value{v.ExactString(): v}, true

// 	case *types.Var:
// 		return scanVar(ident, obj, files, info)
// 	}

// 	return nil, false
// }

// // scanVar inspects the code in the scope of ident, which is a variable,
// // to determine the possible constant values it can have.
// func scanVar(ident *ast.Ident, v *types.Var, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
// 	v = v.Origin()

// 	scope := v.Parent()

// 	node := findSmallestEnclosingNode(files, scope)
// 	if node == nil {
// 		return nil, false
// 	}

// 	// Find all assignments to v within node.
// 	var (
// 		vals     = make(map[string]constant.Value)
// 		complete = true
// 	)

// 	ast.Inspect(node, func(n ast.Node) bool {
// 		if n == nil {
// 			return false
// 		}

// 		switch n := n.(type) {
// 		case *ast.AssignStmt:
// 			vv, ok := scanAssignment(n, v, files, info)
// 			for _, val := range vv {
// 				vals[val.ExactString()] = val
// 			}
// 			complete = complete && ok

// 		case *ast.UnaryExpr:
// 			if n.Op != token.AND {
// 				return true
// 			}
// 			if !exprIsVar(n.X, v, info) {
// 				return true
// 			}
// 			complete = false
// 			// TODO: try to analyze what is done with the address of v

// 		case *ast.ValueSpec:
// 			// Is v on the left-hand side?
// 			found := -1
// 			for i, lhs := range n.Names {
// 				if identIsVar(lhs, v, info) {
// 					found = i
// 					break
// 				}
// 			}
// 			if found < 0 {
// 				return true
// 			}

// 			switch len(n.Values) {
// 			case 0:
// 				// Add the zero value for v to the map.
// 				typ := v.Type().Underlying()
// 				basic, ok := typ.(*types.Basic)
// 				if !ok {
// 					complete = false
// 					return true
// 				}
// 				switch basic.Kind() {
// 				case types.Bool:
// 					v := constant.MakeBool(false)
// 					vals[v.ExactString()] = v

// 				case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
// 					v := constant.MakeInt64(0)
// 					vals[v.ExactString()] = v

// 				case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64:
// 					v := constant.MakeUint64(0)
// 					vals[v.ExactString()] = v

// 				case types.Float32, types.Float64:
// 					v := constant.MakeFloat64(0)
// 					vals[v.ExactString()] = v

// 				case types.Complex64, types.Complex128:
// 					v := constant.MakeImag(constant.MakeInt64(0))
// 					vals[v.ExactString()] = v

// 				case types.String:
// 					v := constant.MakeString("")
// 					vals[v.ExactString()] = v

// 				default:
// 					complete = false
// 				}
// 				return true

// 			case len(n.Names):
// 				rhsVals, ok := Scan(n.Values[found], files, info)
// 				for _, val := range rhsVals {
// 					vals[val.ExactString()] = val
// 				}
// 				complete = complete && ok

// 			default:
// 				// TODO: handle things like nil slices, pointers, etc.?
// 				complete = false
// 				return true
// 			}
// 		}

// 		return true
// 	})

// 	return vals, complete
// }

// func scanAssignment(stmt *ast.AssignStmt, v *types.Var, files []*ast.File, info *types.Info) (map[string]constant.Value, bool) {
// 	// Is v on the left-hand side?
// 	idx := -1
// 	for i, lhs := range stmt.Lhs {
// 		if exprIsVar(lhs, v, info) {
// 			idx = i
// 			break
// 		}
// 	}
// 	if idx < 0 {
// 		return nil, true
// 	}

// 	if stmt.Tok == token.ASSIGN || stmt.Tok == token.DEFINE {
// 		switch len(stmt.Rhs) {
// 		case len(stmt.Lhs):
// 			return Scan(stmt.Rhs[idx], files, info)

// 		case 1:
// 			rhs := ast.Unparen(stmt.Rhs[0])
// 			call, ok := rhs.(*ast.CallExpr)
// 			if !ok {
// 				// TODO: also handle comma-ok forms.
// 				return nil, false
// 			}
// 			return ScanCallResult(call, idx, files, info)
// 		}

// 		return nil, false
// 	}

// 	op, ok := opmap[stmt.Tok]
// 	if !ok { return nil, false }

// 	// xxx bounds checking
// 	lvals, lcomplete := Scan(stmt.Lhs[0], files, info) // xxx infinite regress! maybe split scanAssignment into "scanPlainAssignment" and "scanOpAssignment"?
// 	rvals, rcomplete := Scan(stmt.Rhs[0], files, info)

// 	var (
// 		result = make(map[string]constant.Value)
// 		complete = lcomplete && rcomplete
// 	)

// 	for _, lval := range lvals {
// 		for _, rval := range rvals {
// 			newval := constant.BinaryOp(lval, op, rval)
// 			if newval == nil || newval.Kind() == constant.Unknown {
// 				complete = false
// 				continue
// 			}
// 			result[newval.ExactString()] = newval
// 		}
// 	}

// 		for _, val := range rhsVals {
// 			result[val.ExactString()] = val
// 		}

// 		return result,
// 		complete = complete && rhsComplete

// 	case token.ADD_ASSIGN:
// 		// xxx bounds checking
// 		rhsVals, rhsComplete = Scan(stmt.Rhs[0], files, info)
// 		var newVals []constant.Value
// 		for _, lval := range result {
// 			for _, rval := range rhsVals {
// 				newVals = append(newVals, constant.BinaryOp(lval, token.ADD, rval))

// 				k := lval.Kind()
// 				if k != rval.Kind() { continue }
// 				switch k {
// 				case constant.String:
// 					newVals = append(newVals, constant.BinaryOp(lval, token.ADD, rval))
// 				case constant.Int:
// 				case constant.Float:
// 				case constant.Complex:
// 				}
// 			}
// 		}

// 	default:
// 		// TODO: handle other assignment operators.
// 		complete = false
// 	}

// 	return result, complete
// }

// var (
// 	cmpOps = []token.Token{
// 		token.EQL, token.LSS, token.GTR,
// 		token.NEQ, token.LEQ, token.GEQ,
// 	}

// 	shiftOps = []token.Token{
// 		token.SHL, token.SHR,
// 	}

// 	assignOps = map[token.Token]token.Token{
// 		token.ADD_ASSIGN: token.ADD,
// 		token.SUB_ASSIGN: token.SUB,
// 		token.MUL_ASSIGN: token.MUL,
// 		token.QUO_ASSIGN: token.QUO,
// 		token.REM_ASSIGN: token.REM,
// 		token.AND_ASSIGN:  token.AND,
// 		token.OR_ASSIGN:   token.OR,
// 		token.XOR_ASSIGN:  token.XOR,
// 		token.SHL_ASSIGN:  token.SHL,
// 		token.SHR_ASSIGN:  token.SHR,
// 		token.AND_NOT_ASSIGN: token.AND_NOT,
// 	}
// )

// func exprIsVar(expr ast.Expr, v *types.Var, info *types.Info) bool {
// 	expr = ast.Unparen(expr)
// 	id, ok := expr.(*ast.Ident)
// 	if !ok {
// 		return false
// 	}
// 	return identIsVar(id, v, info)
// }

// func identIsVar(id *ast.Ident, v *types.Var, info *types.Info) bool {
// 	obj := info.ObjectOf(id)
// 	if obj == nil {
// 		return false
// 	}
// 	vv, ok := obj.(*types.Var)
// 	if !ok {
// 		return false
// 	}
// 	return vv.Origin() == v.Origin()
// }

// func findSmallestEnclosingNode(files []*ast.File, node ast.Node) ast.Node {
// 	var result ast.Node

// 	for _, file := range files {
// 		ast.Inspect(file, func(n ast.Node) bool {
// 			if n == nil {
// 				return false
// 			}
// 			if !nodeContains(n, node) {
// 				return false
// 			}
// 			if result == nil || nodeSmaller(n, result) {
// 				result = n
// 			}
// 			return true
// 		})
// 	}

// 	return result
// }

// // Does a contain b?
// func nodeContains(a, b ast.Node) bool {
// 	return a.Pos() <= b.Pos() && a.End() >= b.End()
// }

// // Does a cover a smaller extent than b?
// func nodeSmaller(a, b ast.Node) bool {
// 	return a.Pos() > b.Pos() || a.End() < b.End()
// }
