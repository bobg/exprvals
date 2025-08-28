package exprvals

import (
	"go/ast"
	"go/constant"
	"go/types"
	"iter"
	"maps"
	"slices"

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

func getFuncOrBuiltinForCall(call *ast.CallExpr, pkg *packages.Package) (*types.Func, *types.Builtin) {
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
		return nil, nil
	}

	switch fnObj := fnObj.(type) {
	case *types.Func:
		return fnObj, nil
	case *types.Builtin:
		return nil, fnObj
	case *types.Var:
		// xxx scan var at fnExpr.Pos()
		// xxx if its value set is complete and a single function,
		// xxx that's the answer.
	}

	return nil, nil
}

func getFuncForCall(call *ast.CallExpr, pkg *packages.Package) *types.Func {
	fn, _ := getFuncOrBuiltinForCall(call, pkg)
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

func exprIsVar(expr ast.Expr, v *types.Var, pkg *packages.Package) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	obj := pkg.TypesInfo.ObjectOf(id)
	if obj == nil {
		return false
	}
	ov, ok := obj.(*types.Var)
	if !ok {
		return false
	}
	return ov.Origin() == v.Origin()
}

func isNonLocalExitBuiltin(b *types.Builtin) bool {
	return b.Name() == "panic"
}

var nonLocalExitFuncs = map[string][]string{
	"os":      {"Exit"},
	"testing": {"Fatal", "Fatalf", "FailNow", "SkipNow"},
}

func isNonLocalExitFunc(fn *types.Func) bool {
	if fn == nil {
		return false
	}
	fnpkg := fn.Pkg()
	if fnpkg == nil {
		return false
	}
	fns, ok := nonLocalExitFuncs[fnpkg.Path()]
	if !ok {
		return false
	}
	if slices.Contains(fns, fn.Name()) {
		return true
	}
	return false
}
