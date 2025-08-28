package exprvals

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func ScanCallResult(call *ast.CallExpr, idx int, pkg *packages.Package) (Map, bool) {
	fn := getFuncForCall(call, pkg) // xxx or builtin?
	if fn == nil {
		return nil, false
	}
	sig := fn.Signature()
	if sig == nil {
		return nil, false
	}
	results := sig.Results()
	if results == nil {
		return nil, false
	}
	if idx < 0 || idx >= results.Len() {
		return nil, false
	}
	resultVar := results.At(idx)

	body := getBodyForFunc(fn, pkg)
	if body == nil {
		return nil, false
	}

	sc := newStmtScanner(resultVar, idx, nil)
	sc.blockStmt(body, pkg)

	return sc.vals, sc.complete
}
