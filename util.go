package exprvals

import (
	"go/ast"
	"go/constant"
	"go/types"
	"iter"
	"maps"

	"golang.org/x/tools/go/packages"
)

func isSingleValue(expr ast.Expr, pkg *packages.Package) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return true
	}
	sig := getSignatureForCall(call, pkg)
	if sig == nil {
		return false
	}
	res := sig.Results()
	if res == nil {
		return false
	}
	return res.Len() == 1
}

func getSignatureForCall(call *ast.CallExpr, pkg *packages.Package) *types.Signature {
	funType := pkg.TypesInfo.TypeOf(call.Fun)
	if funType == nil {
		return nil
	}
	sig, ok := funType.Underlying().(*types.Signature)
	if !ok {
		return nil
	}
	return sig
}

func getFuncForCall(call *ast.CallExpr, pkg *packages.Package) *types.Func {
	var (
		fnExpr = ast.Unparen(call.Fun)
		fnObj  types.Object
	)

	switch fnExpr := fnExpr.(type) {
	case *ast.Ident:
		fnObj = pkg.TypesInfo.ObjectOf(fnExpr)

	case *ast.SelectorExpr:
		if sel, ok := pkg.TypesInfo.Selections[fnExpr]; ok {
			fnObj = sel.Obj()
		}
	}

	if fnObj == nil {
		return nil
	}

	fn, ok := fnObj.(*types.Func)
	if !ok {
		return nil
	}
	return fn
}

func getBodyForFunc(fn *types.Func, pkg *packages.Package) *ast.BlockStmt {
	scope := fn.Scope()

	// If pkg does not match fn's pkg, find the right pkg among pkg's imports.

	fnPkg := fn.Pkg()
	if fnPkg == nil {
		return nil
	}

	if fnPkg.Path() != pkg.PkgPath {
		pkg = nil
		for _, ipkg := range pkg.Imports {
			if ipkg.PkgPath == fnPkg.Path() {
				pkg = ipkg
				break
			}
		}
		if pkg == nil {
			return nil
		}
	}

	// Find the smallest *ast.BlockStmt node containing scope.

	var body *ast.BlockStmt
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if n == nil {
				return false
			}

			// Does n contain scope?

			if n.Pos() > scope.Pos() || scope.End() > n.End() {
				return false
			}

			// Is n a block statement?

			bs, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}

			if body == nil || (body.End()-body.Pos()) > (bs.End()-bs.Pos()) {
				body = bs
			}

			return true
		})

		if body != nil {
			break
		}
	}

	return body
}

func anyBoolVals(vals iter.Seq[Value], want bool) bool {
	for v := range vals {
		cv, ok := v.(constant.Value)
		if !ok {
			continue
		}
		if cv.Kind() != constant.Bool {
			continue
		}
		if constant.BoolVal(cv) == want {
			return true
		}
	}
	return false
}

func mergeMaps(a, b Map) Map {
	result := make(Map)
	maps.Copy(result, a)
	maps.Copy(result, b)
	return result
}
