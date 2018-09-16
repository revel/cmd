// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package parser

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*model.MethodSpec

// ProcessSource parses the app controllers directory and
// returns a list of the controller types found.
// Otherwise CompileError if the parsing fails.
func ProcessSource(paths *model.RevelContainer) (*model.SourceInfo, *utils.Error) {
	var (
		srcInfo      *model.SourceInfo
		compileError *utils.Error
	)

	for _, root := range paths.CodePaths {
		rootImportPath := importPathFromPath(root)
		if rootImportPath == "" {
			utils.Logger.Info("Skipping empty code path", "path", root)
			continue
		}

		// Start walking the directory tree.
		_ = utils.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				utils.Logger.Error("Error scanning app source:", "error", err)
				return nil
			}

			if !info.IsDir() || info.Name() == "tmp" {
				return nil
			}

			// Get the import path of the package.
			pkgImportPath := rootImportPath
			if root != path {
				pkgImportPath = rootImportPath + "/" + filepath.ToSlash(path[len(root)+1:])
			}

			// Parse files within the path.
			var pkgs map[string]*ast.Package
			fset := token.NewFileSet()
			pkgs, err = parser.ParseDir(fset, path, func(f os.FileInfo) bool {
				return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
			}, 0)
			if err != nil {
				if errList, ok := err.(scanner.ErrorList); ok {
					var pos = errList[0].Pos
					compileError = &utils.Error{
						SourceType:  ".go source",
						Title:       "Go Compilation Error",
						Path:        pos.Filename,
						Description: errList[0].Msg,
						Line:        pos.Line,
						Column:      pos.Column,
						SourceLines: utils.MustReadLines(pos.Filename),
					}

					errorLink := paths.Config.StringDefault("error.link", "")

					if errorLink != "" {
						compileError.SetLink(errorLink)
					}

					return compileError
				}

				// This is exception, err already checked above. Here just a print
				ast.Print(nil, err)
				utils.Logger.Fatal("Failed to parse dir", "error", err)
			}

			// Skip "main" packages.
			delete(pkgs, "main")

			// If there is no code in this directory, skip it.
			if len(pkgs) == 0 {
				return nil
			}

			// Ignore packages that end with _test
			for i := range pkgs {
				if len(i) > 6 {
					if string(i[len(i)-5:]) == "_test" {
						delete(pkgs, i)
					}
				}
			}

			// There should be only one package in this directory.
			if len(pkgs) > 1 {
				for i := range pkgs {
					println("Found package ", i)
				}
				utils.Logger.Fatal("Most unexpected! Multiple packages in a single directory:", "packages", pkgs)
			}

			var pkg *ast.Package
			for _, v := range pkgs {
				pkg = v
			}

			srcInfo = appendSourceInfo(srcInfo, processPackage(fset, pkgImportPath, path, pkg))
			return nil
		})
	}

	return srcInfo, compileError
}

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

func processPackage(fset *token.FileSet, pkgImportPath, pkgPath string, pkg *ast.Package) *model.SourceInfo {
	var (
		structSpecs     []*model.TypeInfo
		initImportPaths []string

		methodSpecs     = make(methodMap)
		validationKeys  = make(map[string]map[int]string)
		scanControllers = strings.HasSuffix(pkgImportPath, "/controllers") ||
			strings.Contains(pkgImportPath, "/controllers/")
		scanTests = strings.HasSuffix(pkgImportPath, "/tests") ||
			strings.Contains(pkgImportPath, "/tests/")
	)

	// For each source file in the package...
	utils.Logger.Info("Exaimining files in path", "package", pkgPath)
	for fname, file := range pkg.Files {
		// Imports maps the package key to the full import path.
		// e.g. import "sample/app/models" => "models": "sample/app/models"
		imports := map[string]string{}

		// For each declaration in the source file...
		for _, decl := range file.Decls {
			addImports(imports, decl, pkgPath)

			if scanControllers {
				// Match and add both structs and methods
				structSpecs = appendStruct(fname, structSpecs, pkgImportPath, pkg, decl, imports, fset)
				appendAction(fset, methodSpecs, decl, pkgImportPath, pkg.Name, imports)
			} else if scanTests {
				structSpecs = appendStruct(fname, structSpecs, pkgImportPath, pkg, decl, imports, fset)
			}

			// If this is a func... (ignore nil for external (non-Go) function)
			if funcDecl, ok := decl.(*ast.FuncDecl); ok && funcDecl.Body != nil {
				// Scan it for validation calls
				lineKeys := GetValidationKeys(fname, fset, funcDecl, imports)
				if len(lineKeys) > 0 {
					validationKeys[pkgImportPath+"."+getFuncName(funcDecl)] = lineKeys
				}

				// Check if it's an init function.
				if funcDecl.Name.Name == "init" {
					initImportPaths = []string{pkgImportPath}
				}
			}
		}
	}

	// Add the method specs to the struct specs.
	for _, spec := range structSpecs {
		spec.MethodSpecs = methodSpecs[spec.StructName]
	}

	return &model.SourceInfo{
		StructSpecs:     structSpecs,
		ValidationKeys:  validationKeys,
		InitImportPaths: initImportPaths,
	}
}

// getFuncName returns a name for this func or method declaration.
// e.g. "(*Application).SayHello" for a method, "SayHello" for a func.
func getFuncName(funcDecl *ast.FuncDecl) string {
	prefix := ""
	if funcDecl.Recv != nil {
		recvType := funcDecl.Recv.List[0].Type
		if recvStarType, ok := recvType.(*ast.StarExpr); ok {
			prefix = "(*" + recvStarType.X.(*ast.Ident).Name + ")"
		} else {
			prefix = recvType.(*ast.Ident).Name
		}
		prefix += "."
	}
	return prefix + funcDecl.Name.Name
}

func addImports(imports map[string]string, decl ast.Decl, srcDir string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		var pkgAlias string
		if importSpec.Name != nil {
			pkgAlias = importSpec.Name.Name
			if pkgAlias == "_" {
				continue
			}
		}
		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes

		// If the package was not aliased (common case), we have to import it
		// to see what the package name is.
		// TODO: Can improve performance here a lot:
		// 1. Do not import everything over and over again.  Keep a cache.
		// 2. Exempt the standard library; their directories always match the package name.
		// 3. Can use build.FindOnly and then use parser.ParseDir with mode PackageClauseOnly
		if pkgAlias == "" {
			pkg, err := build.Import(fullPath, srcDir, 0)
			if err != nil {
				// We expect this to happen for apps using reverse routing (since we
				// have not yet generated the routes).  Don't log that.
				if !strings.HasSuffix(fullPath, "/app/routes") {
					utils.Logger.Info("Debug: Could not find import:", "path", fullPath)
				}
				continue
			}
			pkgAlias = pkg.Name
		}

		imports[pkgAlias] = fullPath
	}
}

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
				utils.Logger.Warn("Warn: Didn't understand argument '%s' of action %s. Ignoring.", name, getFuncName(funcDecl))
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

// getStructTypeDecl checks if the given decl is a type declaration for a
// struct.  If so, the TypeSpec is returned.
func getStructTypeDecl(decl ast.Decl, fset *token.FileSet) (spec *ast.TypeSpec, found bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.TYPE {
		return
	}

	if len(genDecl.Specs) == 0 {
		utils.Logger.Warn("Warn: Surprising: %s:%d Decl contains no specifications", fset.Position(decl.Pos()).Filename, fset.Position(decl.Pos()).Line)
		return
	}

	spec = genDecl.Specs[0].(*ast.TypeSpec)
	_, found = spec.Type.(*ast.StructType)

	return
}

func importPathFromPath(root string) string {
	if vendorIdx := strings.Index(root, "/vendor/"); vendorIdx != -1 {
		return filepath.ToSlash(root[vendorIdx+8:])
	}
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		srcPath := filepath.Join(gopath, "src")
		if strings.HasPrefix(root, srcPath) {
			return filepath.ToSlash(root[len(srcPath)+1:])
		}
	}

	srcPath := filepath.Join(build.Default.GOROOT, "src", "pkg")
	if strings.HasPrefix(root, srcPath) {
		utils.Logger.Warn("Code path should be in GOPATH, but is in GOROOT:", "path", root)
		return filepath.ToSlash(root[len(srcPath)+1:])
	}

	utils.Logger.Error("Unexpected! Code path is not in GOPATH:", "path", root)
	return ""
}
