package exprvals

import (
	"embed"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type wantPair struct {
	vals     map[string]constant.Value
	complete bool
}

func TestScanVar(t *testing.T) {
	wants := map[string]wantPair{
		"address_taken": wantPair{
			vals:     map[string]constant.Value{`"hello"`: constant.MakeString("hello")},
			complete: false,
		},
		"call_result": wantPair{
			vals:     map[string]constant.Value{`"hello"`: constant.MakeString("hello")},
			complete: true,
		},
		"if_assignment": wantPair{
			vals: map[string]constant.Value{
				`"hello"`:   constant.MakeString("hello"),
				`"goodbye"`: constant.MakeString("goodbye"),
			},
			complete: true,
		},
		"simple_assignment": wantPair{
			vals:     map[string]constant.Value{`"hello"`: constant.MakeString("hello")},
			complete: true,
		},
		"unknown_assignment": wantPair{
			vals:     map[string]constant.Value{`"hello"`: constant.MakeString("hello")},
			complete: false,
		},
		"zero_value": wantPair{
			vals:     map[string]constant.Value{`""`: constant.MakeString("")},
			complete: true,
		},
		"zero_value_int": wantPair{
			vals:     map[string]constant.Value{`0`: constant.MakeInt64(0)},
			complete: true,
		},
		"zero_value_float": wantPair{
			vals:     map[string]constant.Value{`0`: constant.MakeFloat64(0.0)},
			complete: true,
		},
		"zero_value_bool": wantPair{
			vals:     map[string]constant.Value{`false`: constant.MakeBool(false)},
			complete: true,
		},
		"zero_value_complex": wantPair{
			vals:     map[string]constant.Value{`(0 + 0i)`: constant.MakeImag(constant.MakeInt64(0))},
			complete: true,
		},
	}

	const testdata = "testdata/scanvar"

	entries, err := testdataFS.ReadDir(testdata)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		name := entry.Name()
		name = strings.TrimSuffix(name, ".go")
		t.Run(name, func(t *testing.T) {
			src, err := testdataFS.ReadFile(filepath.Join(testdata, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, entry.Name(), src, 0)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{
				Defs:       make(map[*ast.Ident]types.Object),
				Implicits:  make(map[ast.Node]types.Object),
				Scopes:     make(map[ast.Node]*types.Scope),
				Selections: make(map[*ast.SelectorExpr]*types.Selection),
				Types:      make(map[ast.Expr]types.TypeAndValue),
				Uses:       make(map[*ast.Ident]types.Object),
			}
			var conf types.Config
			if _, err := conf.Check("test", fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}

			var ident *ast.Ident
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					return false
				}
				if ident != nil {
					return false
				}
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				if len(ret.Results) != 1 {
					return true
				}
				retval := ast.Unparen(ret.Results[0])
				id, ok := retval.(*ast.Ident)
				if !ok {
					return true
				}
				ident = id
				return false
			})

			if ident == nil {
				t.Fatalf("no single-identifier return value found")
			}

			identObj := info.ObjectOf(ident)
			if identObj == nil {
				t.Fatalf("no object for identifier %s", ident.Name)
			}
			v, ok := identObj.(*types.Var)
			if !ok {
				t.Fatalf("object for identifier %s is a %T, want *types.Var", ident.Name, identObj)
			}

			gotVals, gotComplete := scanVar(ident, v, []*ast.File{file}, info)

			want := wants[name]
			if !reflect.DeepEqual(gotVals, want.vals) {
				t.Errorf("got %v, want %v", gotVals, want.vals)
			}
			if gotComplete != want.complete {
				t.Errorf("got complete = %v, want %v", gotComplete, want.complete)
			}
		})
	}
}

func TestScanCallResult(t *testing.T) {
	wants := map[string]wantPair{
		"simplest": wantPair{
			vals:     map[string]constant.Value{`"hello"`: constant.MakeString("hello")},
			complete: true,
		},
	}

	const testdata = "testdata/scancallresult"

	entries, err := testdataFS.ReadDir(testdata)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		name := entry.Name()
		name = strings.TrimSuffix(name, ".go")
		t.Run(name, func(t *testing.T) {
			src, err := testdataFS.ReadFile(filepath.Join(testdata, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, entry.Name(), src, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			info := &types.Info{
				Defs:       make(map[*ast.Ident]types.Object),
				Implicits:  make(map[ast.Node]types.Object),
				Scopes:     make(map[ast.Node]*types.Scope),
				Selections: make(map[*ast.SelectorExpr]*types.Selection),
				Types:      make(map[ast.Expr]types.TypeAndValue),
				Uses:       make(map[*ast.Ident]types.Object),
			}
			var conf types.Config
			if _, err := conf.Check("test", fset, []*ast.File{file}, info); err != nil {
				t.Fatal(err)
			}

			// Find the last call expression in the file.
			var call *ast.CallExpr
			ast.Inspect(file, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					call = c
				}
				return true
			})
			if call == nil {
				t.Fatal("no call expression found")
			}

			// Find the first comment in the file after the call expression.
			var cg *ast.CommentGroup
			for _, c := range file.Comments {
				if c.Pos() >= call.End() {
					cg = c
					break
				}
			}
			if cg == nil {
				t.Fatalf("no comment group found after call expression %v", call)
			}

			// Parse the comment group to get the index of the result we want.
			const prefix = "// index:"
			idx := -1
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, prefix) {
					continue
				}
				idx, err = strconv.Atoi(c.Text[len(prefix):])
				if err != nil {
					t.Fatal(err)
				}
			}
			if idx < 0 {
				t.Fatal("no index found in comment group")
			}

			gotVals, gotComplete := ScanCallResult(call, idx, []*ast.File{file}, info)
			want := wants[name]
			if !reflect.DeepEqual(gotVals, want.vals) {
				t.Errorf("got %v, want %v", gotVals, want.vals)
			}
			if gotComplete != want.complete {
				t.Errorf("got complete = %v, want %v", gotComplete, want.complete)
			}
		})
	}
}

//go:embed testdata/*
var testdataFS embed.FS
