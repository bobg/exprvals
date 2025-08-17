package exprvals

import (
	"embed"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"strings"
	"testing"
)

func TestScanVar(t *testing.T) {
	wants := map[string]map[string]constant.Value{
		"address_taken": {
			`"hello"`: constant.MakeString("hello"),
		},
		"if_assignment": {
			`"hello"`:   constant.MakeString("hello"),
			`"goodbye"`: constant.MakeString("goodbye"),
		},
		"simple_assignment": {
			`"hello"`: constant.MakeString("hello"),
		},
		"unknown_assignment": {
			`"hello"`: constant.MakeString("hello"),
		},
		"zero_value": {
			`""`: constant.MakeString(""),
		},
		"zero_value_int": {
			`0`: constant.MakeInt64(0),
		},
		"zero_value_float": {
			`0`: constant.MakeFloat64(0.0),
		},
		"zero_value_bool": {
			`false`: constant.MakeBool(false),
		},
		"zero_value_complex": {
			`(0 + 0i)`: constant.MakeImag(constant.MakeInt64(0)),
		},
	}
	wantOKs := map[string]bool{
		"address_taken":      false,
		"if_assignment":      true,
		"simple_assignment":  true,
		"unknown_assignment": false,
		"zero_value":         true,
		"zero_value_int":     true,
		"zero_value_float":   true,
		"zero_value_bool":    true,
		"zero_value_complex": true,
	}

	entries, err := testdata.ReadDir("testdata")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		name := entry.Name()
		name = strings.TrimPrefix(name, "testdata/")
		name = strings.TrimSuffix(name, ".go")
		t.Run(name, func(t *testing.T) {
			src, err := testdata.ReadFile("testdata/" + entry.Name())
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
			conf := types.Config{FakeImportC: true}
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

			got, gotOK := scanVar(ident, v, []*ast.File{file}, info)

			want, wantOK := wants[name], wantOKs[name]
			if !reflect.DeepEqual(got, want) {
				t.Errorf("got %v, want %v", got, want)
			}
			if gotOK != wantOK {
				t.Errorf("got complete = %v, want %v", gotOK, wantOK)
			}
		})
	}
}

//go:embed testdata/*
var testdata embed.FS
