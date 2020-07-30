package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

var TypeExprs = map[string]TypeExpr{
	"int":        TypeExpr{"int", "", 0, true},
	"*int":       TypeExpr{"*int", "", 1, true},
	"[]int":      TypeExpr{"[]int", "", 2, true},
	"...int":     TypeExpr{"[]int", "", 2, true},
	"[]*int":     TypeExpr{"[]*int", "", 3, true},
	"...*int":    TypeExpr{"[]*int", "", 3, true},
	"MyType":     TypeExpr{"MyType", "pkg", 0, true},
	"*MyType":    TypeExpr{"*MyType", "pkg", 1, true},
	"[]MyType":   TypeExpr{"[]MyType", "pkg", 2, true},
	"...MyType":  TypeExpr{"[]MyType", "pkg", 2, true},
	"[]*MyType":  TypeExpr{"[]*MyType", "pkg", 3, true},
	"...*MyType": TypeExpr{"[]*MyType", "pkg", 3, true},
}

func TestTypeExpr(t *testing.T) {
	for str, expected := range TypeExprs {
		typeStr := str
		// Handle arrays and ... myself, since ParseExpr() does not.
		array := strings.HasPrefix(typeStr, "[]")
		if array {
			typeStr = typeStr[2:]
		}

		ellipsis := strings.HasPrefix(typeStr, "...")
		if ellipsis {
			typeStr = typeStr[3:]
		}

		expr, err := parser.ParseExpr(typeStr)
		if err != nil {
			t.Error("Failed to parse test expr:", typeStr)
			continue
		}

		if array {
			expr = &ast.ArrayType{Lbrack: expr.Pos(), Len: nil, Elt: expr}
		}
		if ellipsis {
			expr = &ast.Ellipsis{Ellipsis: expr.Pos(), Elt: expr}
		}

		actual := NewTypeExpr("pkg", expr)
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf("Fail, expected '%v' for '%s', got '%v'\n", expected, str, actual)
		}
	}
}

const testApplication = `
package test

import (
	"os"

	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	myMars "github.com/roblillack/mars"
)

type Hotel struct {
	HotelId          int
	Name, Address    string
	City, State, Zip string
	Country          string
	Price            int
}

type Application struct {
	myMars.Controller
	KnownUser bool
}

type Hotels struct {
	Application
}

type Static struct {
	*myMars.Controller
}

type Bla struct {
	Number int
	Text string
}

type Blurp struct {
	Bla
	Checkbox bool
}

func (blurp Blurp) Index() myMars.Result {
	return nil
}

func (c Hotels) Show(id int) myMars.Result {
	title := "View Hotel"
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.Render(title, hotel)
}

func (c Hotels) Book(id int) myMars.Result {
	hotel := &Hotel{id, "A Hotel", "300 Main St.", "New York", "NY", "10010", "USA", 300}
	return c.RenderJson(hotel)
}

func (c Hotels) Index() myMars.Result {
	return c.RenderText("Hello, World!")
}

func (c Static) Serve(prefix, filepath string) myMars.Result {
	var basePath, dirName string

	if !path.IsAbs(dirName) {
		basePath = BasePath
	}

	fname := path.Join(basePath, prefix, filepath)
	file, err := os.Open(fname)
	if os.IsNotExist(err) {
		return c.NotFound("")
	} else if err != nil {
		myMars.WARN.Printf("Problem opening file (%s): %s ", fname, err)
		return c.NotFound("This was found but not sure why we couldn't open it.")
	}
	return c.RenderFile(file, "")
}
`

func stringSlicesEqual(a, b []string) bool {
	type direction struct {
		Slice []string
		Other []string
	}
	for _, t := range []direction{direction{a, b}, direction{b, a}} {
		for idx, v := range t.Slice {
			if idx >= len(t.Other) || t.Other[idx] != v {
				return false
			}
		}
	}
	return true
}

func (a *MethodArg) Equals(o *MethodArg) bool {
	if a == o {
		return true
	}
	if (a == nil && o != nil) || (o == nil && a != nil) {
		return false
	}

	return a.ImportPath == o.ImportPath && a.Name == o.Name && a.TypeExpr == o.TypeExpr
}

func (s *MethodSpec) Equals(o *MethodSpec) bool {
	if s.Name != o.Name {
		return false
	}

	type direction struct {
		Slice []*MethodArg
		Other []*MethodArg
	}
	for _, t := range []direction{direction{s.Args, o.Args}, direction{o.Args, s.Args}} {
		for idx, v := range t.Slice {
			if idx >= len(t.Other) || !v.Equals(t.Other[idx]) {
				return false
			}
		}
	}
	return true
}

func (i *TypeInfo) Equals(o *TypeInfo) bool {
	if i.ImportPath != o.ImportPath || i.PackageName != o.PackageName || i.StructName != o.StructName {
		return false
	}

	type direction struct {
		Slice []*MethodSpec
		Other []*MethodSpec
	}
	for _, t := range []direction{direction{i.MethodSpecs, o.MethodSpecs}, direction{o.MethodSpecs, i.MethodSpecs}} {
		for idx, v := range t.Slice {
			if idx >= len(t.Other) || !v.Equals(t.Other[idx]) {
				return false
			}
		}
	}
	return true
}

func TestProcessingSource(t *testing.T) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testApplication", testApplication, 0)
	if err != nil {
		t.Fatal(err)
	}

	sourceInfo := ProcessFile(fset, "./test.go", file)
	if n := sourceInfo.PackageName; n != "test" {
		t.Errorf("wrong package name: %s", n)
	}
	if v := sourceInfo.InitImportPaths; !stringSlicesEqual(v, []string{}) {
		t.Errorf("unexpeced import paths: %+v", v)
	}

	if s := sourceInfo.StructSpecs[0]; !s.Equals(&TypeInfo{
		StructName:  "Hotel",
		ImportPath:  "test",
		PackageName: "test",
		MethodSpecs: []*MethodSpec{},
	}) {
		t.Errorf("unexpected struct spec: %+v", s)
	}

	if c := sourceInfo.ControllerSpecs()[0]; !c.Equals(&TypeInfo{
		StructName:  "Application",
		ImportPath:  "test",
		PackageName: "test",
	}) {
		t.Errorf("wrong controller spec for Application controller: %+v", c)
	}

	if c := sourceInfo.ControllerSpecs()[1]; !c.Equals(&TypeInfo{
		StructName:  "Hotels",
		ImportPath:  "test",
		PackageName: "test",
		MethodSpecs: []*MethodSpec{
			&MethodSpec{
				Name: "Show",
				Args: []*MethodArg{
					&MethodArg{
						Name:       "id",
						ImportPath: "",
						TypeExpr:   TypeExpr{"int", "", 0, true},
					},
				},
			},
			&MethodSpec{
				Name: "Book",
				Args: []*MethodArg{
					&MethodArg{
						Name:       "id",
						ImportPath: "",
						TypeExpr:   TypeExpr{"int", "", 0, true},
					},
				},
			},
			&MethodSpec{
				Name: "Index",
			},
		},
	}) {
		t.Errorf("wrong controller spec for Hotels controller: %+v", c)
	}

	if c := sourceInfo.ControllerSpecs()[2]; !c.Equals(&TypeInfo{
		StructName:  "Static",
		ImportPath:  "test",
		PackageName: "test",
		MethodSpecs: []*MethodSpec{
			&MethodSpec{
				Name: "Serve",
				Args: []*MethodArg{
					&MethodArg{
						Name:       "prefix",
						ImportPath: "",
						TypeExpr:   TypeExpr{"string", "", 0, true},
					},
					&MethodArg{
						Name:       "filepath",
						ImportPath: "",
						TypeExpr:   TypeExpr{"string", "", 0, true},
					},
				},
			},
		},
	}) {
		t.Errorf("wrong controller spec for Static controller: %+v", c)
	}
}

func BenchmarkParsingFile(b *testing.B) {
	var fset *token.FileSet
	var file *ast.File

	for n := 0; n < b.N; n++ {
		fset = token.NewFileSet()
		var err error
		file, err = parser.ParseFile(fset, "testApplication", testApplication, 0)
		if err != nil {
			b.Fatal(err)
		}
	}

	ProcessFile(fset, "./test.go", file)
}

func BenchmarkProcessingSource(b *testing.B) {
	var fset *token.FileSet
	var file *ast.File
	fset = token.NewFileSet()
	var err error
	file, err = parser.ParseFile(fset, "testApplication", testApplication, 0)
	if err != nil {
		b.Fatal(err)
	}

	for n := 0; n < b.N; n++ {
		ProcessFile(fset, "./test.go", file)
	}
}
