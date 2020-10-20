// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package parser_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/revel/cmd/model"
	revelParser "github.com/revel/cmd/parser"
)

const validationKeysSource = `
package test

func (c *Application) testFunc(a, b int, user models.User) revel.Result {
	// Line 5
	c.Validation.Required(a)
	c.Validation.Required(a).Message("Error message")
	c.Validation.Required(a).
		Message("Error message")

	// Line 11
	c.Validation.Required(user.Name)
	c.Validation.Required(user.Name).Message("Error message")

	// Line 15
	c.Validation.MinSize(b, 12)
	c.Validation.MinSize(b, 12).Message("Error message")
	c.Validation.MinSize(b,
		12)

	// Line 21
	c.Validation.Required(b == 5)
}

func (m Model) Validate(v *revel.Validation) {
	// Line 26
	v.Required(m.name)
	v.Required(m.name == "something").
		Message("Error Message")
	v.Required(!m.bool)
}
`

var expectedValidationKeys = []map[int]string{
	{
		6:  "a",
		7:  "a",
		8:  "a",
		12: "user.Name",
		13: "user.Name",
		16: "b",
		17: "b",
		19: "b",
		22: "b",
	}, {
		27: "m.name",
		28: "m.name",
		30: "m.bool",
	},
}

// This tests the recording of line number to validation key of the preceding
// example source.
func TestGetValidationKeys(t *testing.T) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "validationKeysSource", validationKeysSource, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Decls) != 2 {
		t.Fatal("Expected 2 decl in the source, found", len(file.Decls))
	}

	for i, decl := range file.Decls {
		lineKeys := revelParser.GetValidationKeys("test", fset, decl.(*ast.FuncDecl), map[string]string{"revel": model.RevelImportPath})
		for k, v := range expectedValidationKeys[i] {
			if lineKeys[k] != v {
				t.Errorf("Not found - %d: %v - Actual Map: %v", k, v, lineKeys)
			}
		}

		if len(lineKeys) != len(expectedValidationKeys[i]) {
			t.Error("Validation key map not the same size as expected:", lineKeys)
		}
	}
}

var TypeExprs = map[string]model.TypeExpr{
	"int":             model.NewTypeExprFromData("int", "", 0, true),
	"*int":            model.NewTypeExprFromData("*int", "", 1, true),
	"[]int":           model.NewTypeExprFromData("[]int", "", 2, true),
	"...int":          model.NewTypeExprFromData("[]int", "", 2, true),
	"[]*int":          model.NewTypeExprFromData("[]*int", "", 3, true),
	"...*int":         model.NewTypeExprFromData("[]*int", "", 3, true),
	"MyType":          model.NewTypeExprFromData("MyType", "pkg", 0, true),
	"*MyType":         model.NewTypeExprFromData("*MyType", "pkg", 1, true),
	"[]MyType":        model.NewTypeExprFromData("[]MyType", "pkg", 2, true),
	"...MyType":       model.NewTypeExprFromData("[]MyType", "pkg", 2, true),
	"[]*MyType":       model.NewTypeExprFromData("[]*MyType", "pkg", 3, true),
	"...*MyType":      model.NewTypeExprFromData("[]*MyType", "pkg", 3, true),
	"map[int]MyType":  model.NewTypeExprFromData("map[int]MyType", "pkg", 8, true),
	"map[int]*MyType": model.NewTypeExprFromData("map[int]*MyType", "pkg", 9, true),
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
			expr = &ast.ArrayType{Lbrack: expr.Pos(), Len: nil, Elt: expr}
		}
		if ellipsis {
			expr = &ast.Ellipsis{Ellipsis: expr.Pos(), Elt: expr}
		}

		actual := model.NewTypeExprFromAst("pkg", expr)
		if !reflect.DeepEqual(expected, actual) {
			t.Error("Fail, expected", expected, ", was", actual)
		}
	}
}
