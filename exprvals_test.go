package exprvals

import (
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestScan(t *testing.T) {
	const testdata = "testdata"

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

			dir := t.TempDir()
			filename := filepath.Join(dir, entry.Name())
			if err := os.WriteFile(filename, src, 0644); err != nil {
				t.Fatal(err)
			}

			gomodpath := filepath.Join(dir, "go.mod")
			f, err := os.Create(gomodpath)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintln(f, "module github.com/bobg/exprvals/test")
			f.Close()

			conf := &packages.Config{
				Dir:  dir,
				Mode: packages.LoadAllSyntax,
				ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
					return parser.ParseFile(fset, filename, src, parser.ParseComments)
				},
			}
			pkgs, err := packages.Load(conf, ".")
			if err != nil {
				t.Fatal(err)
			}

			if len(pkgs) != 1 {
				t.Fatalf("got %d packages, want 1", len(pkgs))
			}

			pkg := pkgs[0]

			if len(pkg.Errors) > 0 {
				t.Fatalf("package load errors: %+v", pkg.Errors)
			}

			if len(pkg.Syntax) != 1 {
				t.Fatalf("got %d syntax files, want 1", len(pkg.Syntax))
			}
			file := pkg.Syntax[0]

			cmap := ast.NewCommentMap(pkg.Fset, file, file.Comments)

			wantVals, wantComplete, expr, err := findWant(file, cmap)
			if err != nil {
				t.Fatal(err)
			}
			sort.Strings(wantVals)

			gotVals, gotComplete := Scan(expr, pkg)
			gotValsList := slices.Collect(maps.Keys(gotVals))
			sort.Strings(gotValsList)

			if !slices.Equal(gotValsList, wantVals) {
				t.Errorf("got vals %v, want %v", gotValsList, wantVals)
			}

			if gotComplete != wantComplete {
				t.Errorf("got complete %v, want %v", gotComplete, wantComplete)
			}

			t.Logf("vals = %v, complete = %v", gotValsList, gotComplete)
		})
	}
}

func findWant(file *ast.File, cmap ast.CommentMap) (vals []string, complete bool, expr ast.Expr, err error) {
OUTER:
	for _, cg := range file.Comments {
		for _, comment := range cg.List {
			text := comment.Text

			const wantPrefix = "// want "
			if !strings.HasPrefix(text, wantPrefix) {
				continue OUTER
			}
			text = strings.TrimPrefix(text, wantPrefix)

			const (
				completePrefix   = "complete"
				incompletePrefix = "incomplete"
			)
			if strings.HasPrefix(text, completePrefix) {
				complete = true
				text = strings.TrimPrefix(text, completePrefix)
			} else if strings.HasPrefix(text, incompletePrefix) {
				text = strings.TrimPrefix(text, incompletePrefix)
			} else {
				err = fmt.Errorf(`malformed "want" comment`)
				return
			}

			text = strings.TrimSpace(text)
			text = strings.TrimPrefix(text, ":")

			text = fmt.Sprintf("[]string{%s}", text)
			expr, err = parser.ParseExpr(text)
			if err != nil {
				return
			}

			lit, ok := expr.(*ast.CompositeLit)
			if !ok {
				err = fmt.Errorf("parsed a %T, want a CompositeLit", expr)
				return
			}
			for _, elt := range lit.Elts {
				basicLit, ok := elt.(*ast.BasicLit)
				if !ok {
					err = fmt.Errorf("parsed a %T, want a string literal", elt)
					return
				}
				if basicLit.Kind != token.STRING {
					err = fmt.Errorf("parsed a %v, want a string literal", basicLit.Kind)
					return
				}
				var s string
				s, err = strconv.Unquote(basicLit.Value)
				if err != nil {
					return
				}
				vals = append(vals, s)
			}

			// Find the associated node.

			var node ast.Node
			for n, cgs := range cmap {
				if slices.ContainsFunc(cgs, func(cg *ast.CommentGroup) bool { return slices.Contains(cg.List, comment) }) {
					node = n
					break
				}
			}
			if node == nil {
				err = fmt.Errorf("could not find node associated with comment")
				return
			}

			// Find the first expr in the node.
			expr = nil
			ast.Inspect(node, func(n ast.Node) bool {
				if n == nil || expr != nil {
					return false
				}
				if e, ok := n.(ast.Expr); ok {
					expr = e
					return false
				}
				return true
			})
			if expr == nil {
				err = fmt.Errorf("could not find expr in node associated with comment")
			}
			return
		}
	}

	err = fmt.Errorf(`no "want" comment found`)
	return
}

//go:embed testdata/*
var testdataFS embed.FS
