package model



// TypeExpr provides a type name that may be rewritten to use a package name.
import (
	"go/ast"
	"fmt"
)

type TypeExpr struct {
	Expr     string // The unqualified type expression, e.g. "[]*MyType"
	PkgName  string // The default package idenifier
	pkgIndex int    // The index where the package identifier should be inserted.
	Valid    bool
}

// TypeName returns the fully-qualified type name for this expression.
// The caller may optionally specify a package name to override the default.
func (e TypeExpr) TypeName(pkgOverride string) string {
	pkgName := FirstNonEmpty(pkgOverride, e.PkgName)
	if pkgName == "" {
		return e.Expr
	}
	return e.Expr[:e.pkgIndex] + pkgName + "." + e.Expr[e.pkgIndex:]
}

// NewTypeExpr returns the syntactic expression for referencing this type in Go.
func NewTypeExpr(pkgName string, expr ast.Expr) TypeExpr {
	error := ""
	switch t := expr.(type) {
	case *ast.Ident:
		if IsBuiltinType(t.Name) {
			pkgName = ""
		}
		return TypeExpr{t.Name, pkgName, 0, true}
	case *ast.SelectorExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{t.Sel.Name, e.Expr, 0, e.Valid}
	case *ast.StarExpr:
		e := NewTypeExpr(pkgName, t.X)
		return TypeExpr{"*" + e.Expr, e.PkgName, e.pkgIndex + 1, e.Valid}
	case *ast.ArrayType:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	case *ast.MapType:
		if identKey, ok := t.Key.(*ast.Ident); ok && IsBuiltinType(identKey.Name) {
			e := NewTypeExpr(pkgName, t.Value)
			return TypeExpr{"map[" + identKey.Name + "]" + e.Expr, e.PkgName, e.pkgIndex + len("map["+identKey.Name+"]"), e.Valid}
		}
		error = fmt.Sprintf("Failed to generate name for Map field :%v. Make sure the field name is valid.", t.Key)
	case *ast.Ellipsis:
		e := NewTypeExpr(pkgName, t.Elt)
		return TypeExpr{"[]" + e.Expr, e.PkgName, e.pkgIndex + 2, e.Valid}
	default:
		error = fmt.Sprintf("Failed to generate name for field: %v Package: %v. Make sure the field name is valid.", expr, pkgName)

	}
	return TypeExpr{Valid: false, Expr:error}
}

var builtInTypes = map[string]struct{}{
	"bool":       {},
	"byte":       {},
	"complex128": {},
	"complex64":  {},
	"error":      {},
	"float32":    {},
	"float64":    {},
	"int":        {},
	"int16":      {},
	"int32":      {},
	"int64":      {},
	"int8":       {},
	"rune":       {},
	"string":     {},
	"uint":       {},
	"uint16":     {},
	"uint32":     {},
	"uint64":     {},
	"uint8":      {},
	"uintptr":    {},
}

// IsBuiltinType checks the given type is built-in types of Go
func IsBuiltinType(name string) bool {
	_, ok := builtInTypes[name]
	return ok
}

// Returns the first non empty string from a list of arguements
func FirstNonEmpty(strs ...string) string {
	for _, str := range strs {
		if len(str) > 0 {
			return str
		}
	}
	return ""
}

