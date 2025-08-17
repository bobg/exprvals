package exprvals

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"
)

func parseAndTypecheck(src string) (*ast.File, *token.FileSet, *types.Info, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	conf := types.Config{Importer: nil, FakeImportC: true}
	_, err = conf.Check("main", fset, []*ast.File{file}, info)
	if err != nil {
		return nil, nil, nil, err
	}
	return file, fset, info, nil
}

func findIdent(file *ast.File, name string) *ast.Ident {
	var found *ast.Ident
	ast.Inspect(file, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if ok && id.Name == name {
			found = id
			return false
		}
		return true
	})
	return found
}

func TestScanVar_SimpleAssignment(t *testing.T) {
	src := `
package main
func f() string {
	x := "hello"
	return x
}
`
	file, _, info, err := parseAndTypecheck(src)
	if err != nil {
		t.Fatalf("parseAndTypecheck: %v", err)
	}
	ident := findIdent(file, "x")
	obj, ok := info.ObjectOf(ident).(*types.Var)
	if !ok {
		t.Fatalf("not a var")
	}
	vals, complete := scanVar(ident, obj, []*ast.File{file}, info)
	want := map[string]constant.Value{`"hello"`: constant.MakeString("hello")}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
	if !complete {
		t.Errorf("expected complete = true")
	}
}

func TestScanVar_IfAssignment(t *testing.T) {
	src := `
package main
func f() string {
	x := "hello"
	if true {
		x = "goodbye"
	}
	return x
}
`
	file, _, info, err := parseAndTypecheck(src)
	if err != nil {
		t.Fatalf("parseAndTypecheck: %v", err)
	}
	ident := findIdent(file, "x")
	obj, ok := info.ObjectOf(ident).(*types.Var)
	if !ok {
		t.Fatalf("not a var")
	}
	vals, complete := scanVar(ident, obj, []*ast.File{file}, info)
	want := map[string]constant.Value{
		`"hello"`:   constant.MakeString("hello"),
		`"goodbye"`: constant.MakeString("goodbye"),
	}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
	if !complete {
		t.Errorf("expected complete = true")
	}
}

func TestScanVar_UnknownAssignment(t *testing.T) {
	src := `
package main
func f(y string) string {
	x := "hello"
	x = y
	return x
}
`
	file, _, info, err := parseAndTypecheck(src)
	if err != nil {
		t.Fatalf("parseAndTypecheck: %v", err)
	}
	ident := findIdent(file, "x")
	obj, ok := info.ObjectOf(ident).(*types.Var)
	if !ok {
		t.Fatalf("not a var")
	}
	vals, complete := scanVar(ident, obj, []*ast.File{file}, info)
	want := map[string]constant.Value{`"hello"`: constant.MakeString("hello")}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
	if complete {
		t.Errorf("expected complete = false")
	}
}

func TestScanVar_AddressTaken(t *testing.T) {
	src := `
package main
func f() string {
	x := "hello"
	g(&x)
	return x
}
func g(*string) {}
`
	file, _, info, err := parseAndTypecheck(src)
	if err != nil {
		t.Fatalf("parseAndTypecheck: %v", err)
	}
	ident := findIdent(file, "x")
	obj, ok := info.ObjectOf(ident).(*types.Var)
	if !ok {
		t.Fatalf("not a var")
	}
	vals, complete := scanVar(ident, obj, []*ast.File{file}, info)
	want := map[string]constant.Value{`"hello"`: constant.MakeString("hello")}
	if !reflect.DeepEqual(vals, want) {
		t.Errorf("got %v, want %v", vals, want)
	}
	if complete {
		t.Errorf("expected complete = false")
	}
}
