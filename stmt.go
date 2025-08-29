package exprvals

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"maps"

	"golang.org/x/tools/go/packages"
)

type stmtScanner struct {
	// inputs
	v      *types.Var
	retIdx int

	// input/output
	vals     Map
	complete bool

	// outputs
	canContinue bool
}

func newStmtScanner(v *types.Var, retIdx int, vals Map) *stmtScanner {
	result := make(Map)
	maps.Copy(result, vals)

	return &stmtScanner{
		v:           v,
		retIdx:      retIdx,
		vals:        vals,
		complete:    true,
		canContinue: true,
	}
}

func (s *stmtScanner) dup() *stmtScanner {
	vals := make(Map)
	maps.Copy(vals, s.vals)

	return &stmtScanner{
		v:           s.v,
		retIdx:      s.retIdx,
		vals:        s.vals,
		complete:    s.complete,
		canContinue: true,
	}
}

func (s *stmtScanner) stmtList(stmts []ast.Stmt, pkg *packages.Package) {
	for _, stmt := range stmts {
		if !s.canContinue {
			return
		}
		s.stmt(stmt, pkg)
	}
}

// xxx
// Need a mechanism for automatically creating subscanners for new scopes
// and merging results from them.
//
// Per https://pkg.go.dev/go/types#Info:
//
// The following node types may appear in Scopes:
//
//     *ast.File
//     *ast.FuncType
//     *ast.TypeSpec
//     *ast.BlockStmt
//     *ast.IfStmt
//     *ast.SwitchStmt
//     *ast.TypeSwitchStmt
//     *ast.CaseClause
//     *ast.CommClause
//     *ast.ForStmt
//     *ast.RangeStmt

func (s *stmtScanner) stmt(stmt ast.Stmt, pkg *packages.Package) {
	if stmt == nil {
		return
	}

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		s.assignStmt(stmt, pkg)

	case *ast.BlockStmt:
		s.blockStmt(stmt, pkg)

	case *ast.BranchStmt:
		s.branchStmt(stmt, pkg)

	case *ast.DeclStmt:
		s.declStmt(stmt, pkg)

	case *ast.DeferStmt:
		s.deferStmt(stmt, pkg)

	case *ast.EmptyStmt:
		// do nothing

	case *ast.ExprStmt:
		s.exprStmt(stmt, pkg)

	case *ast.ForStmt:
		s.forStmt(stmt, pkg)

	case *ast.GoStmt:
		s.goStmt(stmt, pkg)

	case *ast.IfStmt:
		s.ifStmt(stmt, pkg)

	case *ast.IncDecStmt:
		s.incDecStmt(stmt, pkg)

	case *ast.LabeledStmt:
		s.stmt(stmt.Stmt, pkg)

	case *ast.RangeStmt:
		s.rangeStmt(stmt, pkg)

	case *ast.ReturnStmt:
		s.returnStmt(stmt, pkg)

	case *ast.SelectStmt:
		s.selectStmt(stmt, pkg)

	case *ast.SendStmt:
		s.sendStmt(stmt, pkg)

	case *ast.SwitchStmt:
		s.switchStmt(stmt, pkg)

	case *ast.TypeSwitchStmt:
		s.typeSwitchStmt(stmt, pkg)

	default:
		s.complete = false
	}
}

func (s *stmtScanner) assignStmt(stmt *ast.AssignStmt, pkg *packages.Package) {
	if stmt == nil {
		return
	}

	idx := -1
	for i, lhs := range stmt.Lhs {
		if exprIsVar(lhs, s.v, pkg) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	var (
		rhsVals     Map
		rhsComplete bool
	)

	switch len(stmt.Rhs) {
	case len(stmt.Lhs):
		rhsVals, rhsComplete = Scan(stmt.Rhs[idx], pkg)

	case 1:
		rhs := ast.Unparen(stmt.Rhs[0])
		switch rhs := rhs.(type) {
		case *ast.CallExpr:
			rhsVals, rhsComplete = ScanCallResult(rhs, idx, pkg)
		}

	default:
		s.complete = false
		return
	}

	switch stmt.Tok {
	case token.ASSIGN, token.DEFINE:
		s.vals = rhsVals
		s.complete = rhsComplete
		return
	}

	op, ok := assignOps[stmt.Tok]
	if !ok {
		s.complete = false
		return
	}

	s.vals, s.complete = scanBinaryExprWithLHS(s.vals, s.complete, op, stmt.Rhs[idx], pkg)
}

func (s *stmtScanner) blockStmt(stmt *ast.BlockStmt, pkg *packages.Package) {
	if stmt == nil {
		return
	}
	s.stmtList(stmt.List, pkg)
}

func (s *stmtScanner) branchStmt(stmt *ast.BranchStmt, pkg *packages.Package) {
	s.canContinue = false // true even for "fallthrough," since no statements may follow it in a block
}

func (s *stmtScanner) declStmt(stmt *ast.DeclStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) deferStmt(stmt *ast.DeferStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) exprStmt(stmt *ast.ExprStmt, pkg *packages.Package) {
	if !s.canContinue {
		return
	}

	// A call to panic here is a non-local exit.
	// Likewise for some stdlib functions, like os.Exit and log.Fatal.

	expr := ast.Unparen(stmt.X)
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	s.callExpr(call, pkg)
}

func (s *stmtScanner) forStmt(stmt *ast.ForStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) goStmt(stmt *ast.GoStmt, pkg *packages.Package) {
	s.callExpr(stmt.Call, pkg)
	s.canContinue = true // go starts a separate goroutine
}

func (s *stmtScanner) callExpr(call *ast.CallExpr, pkg *packages.Package) {
	fn, bi := getFuncOrBuiltinForCall(call, pkg)
	if fn != nil {
		s.canContinue = !isNonLocalExitFunc(fn)
	} else if bi != nil {
		s.canContinue = !isNonLocalExitBuiltin(bi)
	}

	// xxx handle builtin

	if fn != nil {
		body := getBodyForFunc(fn, pkg)
		if body == nil {
			return
		}
		sub := s.dup()
		sub.blockStmt(body, pkg)
	}
}

func (s *stmtScanner) ifStmt(stmt *ast.IfStmt, pkg *packages.Package) {
	s.stmt(stmt.Init, pkg)

	condVals, condComplete := Scan(stmt.Cond, pkg)

	var (
		canBeTrue  = !condComplete || anyBoolVals(maps.Values(condVals), true)
		canBeFalse = !condComplete || anyBoolVals(maps.Values(condVals), false)
	)

	if canBeTrue {
		if !canBeFalse {
			s.blockStmt(stmt.Body, pkg)
			return
		}

		// canBeTrue && canBeFalse

		if stmt.Else == nil {
			s.stmt(stmt.Body, pkg)
			s.canContinue = true
			return
		}

		var (
			thenScanner = s.dup()
			elseScanner = s.dup()
		)

		thenScanner.blockStmt(stmt.Body, pkg)
		elseScanner.stmt(stmt.Else, pkg)
		s.vals = mergeMaps(thenScanner.vals, elseScanner.vals)
		s.complete = thenScanner.complete && elseScanner.complete
		s.canContinue = thenScanner.canContinue || elseScanner.canContinue
		return
	}

	// !canBeTrue

	if canBeFalse {
		s.stmt(stmt.Else, pkg)
		return
	}
}

func (s *stmtScanner) incDecStmt(stmt *ast.IncDecStmt, pkg *packages.Package) {
	if !exprIsVar(stmt.X, s.v, pkg) {
		return
	}

	var delta int64 = 1
	if stmt.Tok == token.DEC {
		delta = -1
	}
	incdec := constant.MakeInt64(delta)
	newVals := make(Map)
	for _, val := range s.vals {
		cv, ok := val.(constant.Value)
		if !ok {
			s.complete = false
			continue
		}
		newVal := constant.BinaryOp(cv, token.ADD, incdec)
		if newVal.Kind() == constant.Unknown {
			s.complete = false
			continue
		}
		newVals[newVal.ExactString()] = newVal
	}
	s.vals = newVals
}

func (s *stmtScanner) rangeStmt(stmt *ast.RangeStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) returnStmt(stmt *ast.ReturnStmt, pkg *packages.Package) {
	s.canContinue = false

	if s.retIdx < 0 || s.retIdx >= len(stmt.Results) {
		return
	}

	s.vals, s.complete = Scan(stmt.Results[s.retIdx], pkg)
}

func (s *stmtScanner) selectStmt(stmt *ast.SelectStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) sendStmt(stmt *ast.SendStmt, pkg *packages.Package) {
	// xxx
}

func (s *stmtScanner) switchStmt(stmt *ast.SwitchStmt, pkg *packages.Package) {
	s.stmt(stmt.Init, pkg)

	var (
		tagVals     Map
		tagComplete bool
	)
	if stmt.Tag == nil {
		trueVal := constant.MakeBool(true)
		tagVals = Map{trueVal.ExactString(): trueVal}
		tagComplete = true
	} else {
		tagVals, tagComplete = Scan(stmt.Tag, pkg)
	}

	type clauseInfo struct {
		scanner      *stmtScanner
		fallsThrough bool
		isDefault    bool
	}
	var clauses []*clauseInfo // only matchable clauses (including those reached via fallthrough)

	for _, stmt := range stmt.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			s.complete = false
			continue
		}

		info := &clauseInfo{
			isDefault: cc.List == nil,
		}

		for i := len(cc.Body) - 1; i >= 0; i-- {
			stmt := cc.Body[i]
			if stmt == nil {
				continue
			}
			if _, ok := stmt.(*ast.EmptyStmt); ok {
				continue
			}
			if b, ok := stmt.(*ast.BranchStmt); ok && b.Tok == token.FALLTHROUGH {
				info.fallsThrough = true
			}
			break
		}

		canMatch := info.isDefault || !tagComplete // TODO: Unless we can prove otherwise (because tagComplete is true and the other cases cover all possible values of the tag expr).
		if !canMatch && len(clauses) > 1 {
			prev := clauses[len(clauses)-1]
			canMatch = prev.fallsThrough
		}

		if !canMatch {
		OUTER:
			for _, expr := range cc.List {
				exprVals, exprComplete := Scan(expr, pkg)
				if !exprComplete {
					canMatch = true
					break
				}
				for _, tagVal := range tagVals {
					cv, ok := tagVal.(constant.Value)
					if !ok {
						canMatch = true
						break OUTER
					}
					for _, exprVal := range exprVals {
						cv2, ok := exprVal.(constant.Value)
						if !ok || constant.Compare(cv, token.EQL, cv2) {
							canMatch = true
							break OUTER
						}
					}
				}
			}
		}

		if !canMatch {
			continue
		}

		sub := s.dup()
		sub.stmtList(cc.Body, pkg)

		info.scanner = sub

		clauses = append(clauses, info)
	}

	s.vals = make(Map)
	s.complete = true

	var hasDefault, canContinue bool

	for _, info := range clauses {
		if info.isDefault {
			hasDefault = true
		}
		sub := info.scanner
		s.vals = mergeMaps(s.vals, sub.vals)
		s.complete = s.complete && sub.complete
		canContinue = canContinue || sub.canContinue
	}
	s.canContinue = canContinue || !hasDefault
}

func (s *stmtScanner) typeSwitchStmt(stmt *ast.TypeSwitchStmt, pkg *packages.Package) {
	// xxx
}
