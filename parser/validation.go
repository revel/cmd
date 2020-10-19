package parser

import (
	"go/ast"
	"go/token"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

// Scan app source code for calls to X.Y(), where X is of type *Validation.
//
// Recognize these scenarios:
// - "Y" = "Validation" and is a member of the receiver.
//   (The common case for inline validation)
// - "X" is passed in to the func as a parameter.
//   (For structs implementing Validated)
//
// The line number to which a validation call is attributed is that of the
// surrounding ExprStmt.  This is so that it matches what runtime.Callers()
// reports.
//
// The end result is that we can set the default validation key for each call to
// be the same as the local variable.
func GetValidationKeys(fname string, fset *token.FileSet, funcDecl *ast.FuncDecl, imports map[string]string) map[int]string {
	var (
		lineKeys = make(map[int]string)

		// Check the func parameters and the receiver's members for the *revel.Validation type.
		validationParam = getValidationParameter(funcDecl, imports)
	)

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		// e.g. c.Validation.Required(arg) or v.Required(arg)
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		// e.g. c.Validation.Required or v.Required
		funcSelector, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		switch x := funcSelector.X.(type) {
		case *ast.SelectorExpr: // e.g. c.Validation
			if x.Sel.Name != "Validation" {
				return true
			}

		case *ast.Ident: // e.g. v
			if validationParam == nil || x.Obj != validationParam {
				return true
			}

		default:
			return true
		}

		if len(callExpr.Args) == 0 {
			return true
		}

		// Given the validation expression, extract the key.
		key := callExpr.Args[0]
		switch expr := key.(type) {
		case *ast.BinaryExpr:
			// If the argument is a binary expression, take the first expression.
			// (e.g. c.Validation.Required(myName != ""))
			key = expr.X
		case *ast.UnaryExpr:
			// If the argument is a unary expression, drill in.
			// (e.g. c.Validation.Required(!myBool)
			key = expr.X
		case *ast.BasicLit:
			// If it's a literal, skip it.
			return true
		}

		if typeExpr := model.NewTypeExprFromAst("", key); typeExpr.Valid {
			lineKeys[fset.Position(callExpr.End()).Line] = typeExpr.TypeName("")
		} else {
			utils.Logger.Error("Error: Failed to generate key for field validation. Make sure the field name is valid.", "file", fname,
				"line", fset.Position(callExpr.End()).Line, "function", funcDecl.Name.String())
		}
		return true
	})

	return lineKeys
}

// Check to see if there is a *revel.Validation as an argument.
func getValidationParameter(funcDecl *ast.FuncDecl, imports map[string]string) *ast.Object {
	for _, field := range funcDecl.Type.Params.List {
		starExpr, ok := field.Type.(*ast.StarExpr) // e.g. *revel.Validation
		if !ok {
			continue
		}

		selExpr, ok := starExpr.X.(*ast.SelectorExpr) // e.g. revel.Validation
		if !ok {
			continue
		}

		xIdent, ok := selExpr.X.(*ast.Ident) // e.g. rev
		if !ok {
			continue
		}

		if selExpr.Sel.Name == "Validation" && imports[xIdent.Name] == model.RevelImportPath {
			return field.Names[0].Obj
		}
	}
	return nil
}
