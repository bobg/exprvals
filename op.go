package exprvals

import (
	"go/ast"
	"go/constant"
	"go/token"
	"math"

	"golang.org/x/tools/go/packages"
)

func scanUnaryExpr(u *ast.UnaryExpr, pkg *packages.Package) (Map, bool) {
	switch u.Op {
	case token.MUL:
		vals, complete := Scan(u.X, pkg)
		result := make(Map)
		for _, v := range vals {
			p, ok := v.(Pointer)
			if !ok {
				complete = false
				continue
			}
			result[p.Elem.ExactString()] = p.Elem
		}
		return result, complete

	case token.AND:
		vals, complete := Scan(u.X, pkg)
		result := make(Map)
		for _, v := range vals {
			p := Pointer{Elem: v}
			result[p.ExactString()] = p
		}
		return result, complete

	default:
		vals, complete := Scan(u.X, pkg)
		result := make(Map)
		for _, v := range vals {
			cv, ok := v.(constant.Value)
			if !ok {
				complete = false
				continue
			}
			vv := constant.UnaryOp(u.Op, cv, 64)
			if vv.Kind() == constant.Unknown {
				complete = false
				continue
			}
			result[vv.ExactString()] = vv
		}
		return result, complete
	}
}

func scanBinaryExpr(b *ast.BinaryExpr, pkg *packages.Package) (Map, bool) {
	lvals, lcomplete := Scan(b.X, pkg)

	return scanBinaryExprWithLHS(lvals, lcomplete, b.Op, b.Y, pkg)
}

func scanBinaryExprWithLHS(lvals Map, lcomplete bool, op token.Token, rhs ast.Expr, pkg *packages.Package) (Map, bool) {
	rvals, rcomplete := Scan(rhs, pkg)
	complete := lcomplete && rcomplete
	result := make(Map)

	for _, lv := range lvals {
		lcv, ok := lv.(constant.Value)
		if !ok {
			complete = false
			continue
		}
		for _, rv := range rvals {
			rcv, ok := rv.(constant.Value)
			if !ok {
				complete = false
				continue
			}

			var vv constant.Value

			// Choose the right operation based on the operator:
			// constant.BinaryOp, constant.Shift, or constant.Compare.

			switch op {
			case token.SHL, token.SHR:
				if lcv.Kind() != constant.Int {
					complete = false
					continue
				}
				rint, ok := constant.Uint64Val(rcv)
				if !ok {
					complete = false
					continue
				}
				if rint > math.MaxUint {
					complete = false
					continue
				}
				vv = constant.Shift(lcv, op, uint(rint))

			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
				cmp := constant.Compare(lcv, op, rcv)
				vv = constant.MakeBool(cmp)

			default:
				vv = constant.BinaryOp(lcv, op, rcv)
			}

			if vv.Kind() == constant.Unknown {
				complete = false
				continue
			}

			result[vv.ExactString()] = vv
		}
	}
	return result, complete
}

var assignOps = map[token.Token]token.Token{
	token.ADD_ASSIGN:     token.ADD,
	token.SUB_ASSIGN:     token.SUB,
	token.MUL_ASSIGN:     token.MUL,
	token.QUO_ASSIGN:     token.QUO,
	token.REM_ASSIGN:     token.REM,
	token.AND_ASSIGN:     token.AND,
	token.OR_ASSIGN:      token.OR,
	token.XOR_ASSIGN:     token.XOR,
	token.SHL_ASSIGN:     token.SHL,
	token.SHR_ASSIGN:     token.SHR,
	token.AND_NOT_ASSIGN: token.AND_NOT,
}
