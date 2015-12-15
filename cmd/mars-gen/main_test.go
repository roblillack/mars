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
	for typeStr, expected := range TypeExprs {
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
			expr = &ast.ArrayType{expr.Pos(), nil, expr}
		}
		if ellipsis {
			expr = &ast.Ellipsis{expr.Pos(), expr}
		}

		actual := NewTypeExpr("pkg", expr)
		if !reflect.DeepEqual(expected, actual) {
			t.Error("Fail, expected", expected, ", was", actual)
		}
	}
}

const testApplication = `
package test

import (
	"os"
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

func TestProcessingSource(t *testing.T) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "testApplication", testApplication, 0)
	if err != nil {
		t.Fatal(err)
	}

	sourceInfo := ProcessFile(fset, "./test.go", file)
	t.Log(sourceInfo.ControllerSpecs())
}
