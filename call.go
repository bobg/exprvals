package exprvals

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func ScanCallResult(call *ast.CallExpr, idx int, pkg *packages.Package) (Map, bool) {
	fn := getFuncForCall(call, pkg)
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

	var (
		result   = make(Map)
		complete = true
	)

	// xxx If resultVar is a named return value,
	// add the zero value of its type to result.

	for _, stmt := range body.List {
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
		case *ast.IncDecStmt:
		case *ast.ReturnStmt:
		}
	}
}
