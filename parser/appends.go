package parser

import (
	"go/ast"
	"github.com/revel/cmd/utils"
	"github.com/revel/cmd/model"
	"go/token"
)

// If this Decl is a struct type definition, it is summarized and added to specs.
// Else, specs is returned unchanged.
func appendStruct(fileName string, specs []*model.TypeInfo, pkgImportPath string, pkg *ast.Package, decl ast.Decl, imports map[string]string, fset *token.FileSet) []*model.TypeInfo {
	// Filter out non-Struct type declarations.
	spec, found := getStructTypeDecl(decl, fset)
	if !found {
		return specs
	}

	structType := spec.Type.(*ast.StructType)

	// At this point we know it's a type declaration for a struct.
	// Fill in the rest of the info by diving into the fields.
	// Add it provisionally to the Controller list -- it's later filtered using field info.
	controllerSpec := &model.TypeInfo{
		StructName:  spec.Name.Name,
		ImportPath:  pkgImportPath,
		PackageName: pkg.Name,
	}

	for _, field := range structType.Fields.List {
		// If field.Names is set, it's not an embedded type.
		if field.Names != nil {
			continue
		}

		// A direct "sub-type" has an ast.Field as either:
		//   Ident { "AppController" }
		//   SelectorExpr { "rev", "Controller" }
		// Additionally, that can be wrapped by StarExprs.
		fieldType := field.Type
		pkgName, typeName := func() (string, string) {
			// Drill through any StarExprs.
			for {
				if starExpr, ok := fieldType.(*ast.StarExpr); ok {
					fieldType = starExpr.X
					continue
				}
				break
			}

			// If the embedded type is in the same package, it's an Ident.
			if ident, ok := fieldType.(*ast.Ident); ok {
				return "", ident.Name
			}

			if selectorExpr, ok := fieldType.(*ast.SelectorExpr); ok {
				if pkgIdent, ok := selectorExpr.X.(*ast.Ident); ok {
					return pkgIdent.Name, selectorExpr.Sel.Name
				}
			}
			return "", ""
		}()

		// If a typename wasn't found, skip it.
		if typeName == "" {
			continue
		}

		// Find the import path for this type.
		// If it was referenced without a package name, use the current package import path.
		// Else, look up the package's import path by name.
		var importPath string
		if pkgName == "" {
			importPath = pkgImportPath
		} else {
			var ok bool
			if importPath, ok = imports[pkgName]; !ok {
				utils.Logger.Error("Error: Failed to find import path for ", "package", pkgName, "type", typeName)
				continue
			}
		}

		controllerSpec.EmbeddedTypes = append(controllerSpec.EmbeddedTypes, &model.EmbeddedTypeName{
			ImportPath: importPath,
			StructName: typeName,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendAction(fset *token.FileSet, mm methodMap, decl ast.Decl, pkgImportPath, pkgName string, imports map[string]string) {
	// Func declaration?
	funcDecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		return
	}

	// Have a receiver?
	if funcDecl.Recv == nil {
		return
	}

	// Is it public?
	if !funcDecl.Name.IsExported() {
		return
	}

	// Does it return a Result?
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return
	}
	selExpr, ok := funcDecl.Type.Results.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if selExpr.Sel.Name != "Result" {
		return
	}
	if pkgIdent, ok := selExpr.X.(*ast.Ident); !ok || imports[pkgIdent.Name] != model.RevelImportPath {
		return
	}

	method := &model.MethodSpec{
		Name: funcDecl.Name.Name,
	}

	// Add a description of the arguments to the method.
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			var importPath string
			typeExpr := model.NewTypeExprFromAst(pkgName, field.Type)
			if !typeExpr.Valid {
				utils.Logger.Warnf("Warn: Didn't understand argument '%s' of action %s. Ignoring.", name, getFuncName(funcDecl))
				return // We didn't understand one of the args.  Ignore this action.
			}
			// Local object
			if typeExpr.PkgName == pkgName {
				importPath = pkgImportPath
			} else if typeExpr.PkgName != "" {
				var ok bool
				if importPath, ok = imports[typeExpr.PkgName]; !ok {
					utils.Logger.Fatalf("Failed to find import for arg of type: %s , %s", typeExpr.PkgName, typeExpr.TypeName(""))
				}
			}
			method.Args = append(method.Args, &model.MethodArg{
				Name:       name.Name,
				TypeExpr:   typeExpr,
				ImportPath: importPath,
			})
		}
	}

	// Add a description of the calls to Render from the method.
	// Inspect every node (e.g. always return true).
	method.RenderCalls = []*model.MethodCall{}
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		// Is it a function call?
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Is it calling (*Controller).Render?
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// The type of the receiver is not easily available, so just store every
		// call to any method called Render.
		if selExpr.Sel.Name != "Render" {
			return true
		}

		// Add this call's args to the renderArgs.
		pos := fset.Position(callExpr.Lparen)
		methodCall := &model.MethodCall{
			Line:  pos.Line,
			Names: []string{},
		}
		for _, arg := range callExpr.Args {
			argIdent, ok := arg.(*ast.Ident)
			if !ok {
				continue
			}
			methodCall.Names = append(methodCall.Names, argIdent.Name)
		}
		method.RenderCalls = append(method.RenderCalls, methodCall)
		return true
	})

	var recvTypeName string
	var recvType = funcDecl.Recv.List[0].Type
	if recvStarType, ok := recvType.(*ast.StarExpr); ok {
		recvTypeName = recvStarType.X.(*ast.Ident).Name
	} else {
		recvTypeName = recvType.(*ast.Ident).Name
	}

	mm[recvTypeName] = append(mm[recvTypeName], method)
}

// Combine the 2 source info models into one
func appendSourceInfo(srcInfo1, srcInfo2 *model.SourceInfo) *model.SourceInfo {
	if srcInfo1 == nil {
		return srcInfo2
	}

	srcInfo1.StructSpecs = append(srcInfo1.StructSpecs, srcInfo2.StructSpecs...)
	srcInfo1.InitImportPaths = append(srcInfo1.InitImportPaths, srcInfo2.InitImportPaths...)
	for k, v := range srcInfo2.ValidationKeys {
		if _, ok := srcInfo1.ValidationKeys[k]; ok {
			utils.Logger.Warn("Warn: Key conflict when scanning validation calls:", "key", k)
			continue
		}
		srcInfo1.ValidationKeys[k] = v
	}
	return srcInfo1
}
