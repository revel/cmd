package harness

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"github.com/robfig/revel"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// SourceInfo is the top-level struct containing all extracted information
// about the app source code, used to generate main.go.
type SourceInfo struct {
	// ControllerSpecs lists type info for all structs found under
	// app/controllers/... that embed (directly or indirectly) rev.Controller.
	ControllerSpecs []*TypeInfo
	// ValidationKeys provides a two-level lookup.  The keys are:
	// 1. The fully-qualified function name,
	//    e.g. "github.com/robfig/revel/samples/chat/app/controllers.(*Application).Action"
	// 2. Within that func's file, the line number of the (overall) expression statement.
	//    e.g. the line returned from runtime.Caller()
	// The result of the lookup the name of variable being validated.
	ValidationKeys map[string]map[int]string
	// UnitTests and FunctionalTests list the types that constitute the app test suite.
	UnitTests, FunctionalTests []*TypeInfo
}

// TypeInfo summarizes information about a struct type in the app source code.
type TypeInfo struct {
	PackageName string // e.g. "controllers"
	StructName  string // e.g. "Application"
	ImportPath  string // e.g. "github.com/robfig/revel/samples/chat/app/controllers"
	MethodSpecs []*MethodSpec

	// Used internally to identify controllers that indirectly embed *rev.Controller.
	embeddedTypes []*embeddedTypeName
}

// methodCall describes a call to c.Render(..)
// It documents the argument names used, in order to propagate them to RenderArgs.
type methodCall struct {
	Path  string // e.g. "myapp/app/controllers.(*Application).Action"
	Line  int
	Names []string
}

type MethodSpec struct {
	Name        string        // Name of the method, e.g. "Index"
	Args        []*MethodArg  // Argument descriptors
	RenderCalls []*methodCall // Descriptions of Render() invocations from this Method.
}

type MethodArg struct {
	Name       string // Name of the argument.
	TypeName   string // The name of the type, e.g. "int", "*pkg.UserType"
	ImportPath string // If the arg is of an imported type, this is the import path.
}

type embeddedTypeName struct {
	PackageName, StructName string
}

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*MethodSpec

// Parse the app controllers directory and return a list of the controller types found.
// Returns a CompileError if the parsing fails.
func ProcessSource() (*SourceInfo, *rev.Error) {
	root := rev.BasePath
	var (
		srcInfo      *SourceInfo
		compileError *rev.Error
	)

	// Start walking the directory tree.
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Error scanning app source:", err)
			return nil
		}

		if !info.IsDir() || path == root {
			return nil
		}

		if info.Name() == "tmp" {
			return nil
		}

		// Get the import path of the package.
		pkgImportPath := rev.ImportPath + "/" + filepath.ToSlash(path[len(root)+1:])

		// Parse files within the path.
		var pkgs map[string]*ast.Package
		fset := token.NewFileSet()
		pkgs, err = parser.ParseDir(fset, path, func(f os.FileInfo) bool {
			return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
		}, 0)
		if err != nil {
			if errList, ok := err.(scanner.ErrorList); ok {
				var pos token.Position = errList[0].Pos
				compileError = &rev.Error{
					SourceType:  ".go source",
					Title:       "Go Compilation Error",
					Path:        pos.Filename,
					Description: errList[0].Msg,
					Line:        pos.Line,
					Column:      pos.Column,
					SourceLines: rev.MustReadLines(pos.Filename),
				}
				return compileError
			}
			ast.Print(nil, err)
			log.Fatalf("Failed to parse dir: %s", err)
		}

		// Skip "main" packages.
		delete(pkgs, "main")

		// If there is no code in this directory, skip it.
		if len(pkgs) == 0 {
			return nil
		}

		// There should be only one package in this directory.
		if len(pkgs) > 1 {
			log.Println("Most unexpected! Multiple packages in a single directory:", pkgs)
		}

		var pkg *ast.Package
		for _, v := range pkgs {
			pkg = v
		}

		srcInfo = appendSourceInfo(srcInfo, processPackage(fset, pkgImportPath, pkg))
		return nil
	})

	return srcInfo, compileError
}

func appendSourceInfo(srcInfo1, srcInfo2 *SourceInfo) *SourceInfo {
	if srcInfo1 == nil {
		return srcInfo2
	}

	srcInfo1.ControllerSpecs = append(srcInfo1.ControllerSpecs, srcInfo2.ControllerSpecs...)
	srcInfo1.UnitTests = append(srcInfo1.UnitTests, srcInfo2.UnitTests...)
	srcInfo1.FunctionalTests = append(srcInfo1.FunctionalTests, srcInfo2.FunctionalTests...)
	for k, v := range srcInfo2.ValidationKeys {
		if _, ok := srcInfo1.ValidationKeys[k]; ok {
			log.Println("Key conflict when scanning validation calls:", k)
			continue
		}
		srcInfo1.ValidationKeys[k] = v
	}
	return srcInfo1
}

func processPackage(fset *token.FileSet, pkgImportPath string, pkg *ast.Package) *SourceInfo {
	var structSpecs []*TypeInfo
	validationKeys := make(map[string]map[int]string)
	methodSpecs := make(methodMap)
	scanControllers := strings.HasSuffix(pkgImportPath, "/controllers") ||
		strings.Contains(pkgImportPath, "/controllers/")
	scanTests := strings.HasSuffix(pkgImportPath, "/tests") ||
		strings.Contains(pkgImportPath, "/tests/")

	// For each source file in the package...
	for _, file := range pkg.Files {

		// Imports maps the package key to the full import path.
		// e.g. import "sample/app/models" => "models": "sample/app/models"
		imports := map[string]string{}

		// For each declaration in the source file...
		for _, decl := range file.Decls {

			if scanControllers {
				// Match and add both structs and methods
				addImports(imports, decl)
				structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl)
				appendAction(fset, methodSpecs, decl, pkg.Name, imports)
			} else if scanTests {
				structSpecs = appendStruct(structSpecs, pkgImportPath, pkg, decl)
			}

			// If this is a func, scan it for validation calls.
			if funcDecl, ok := decl.(*ast.FuncDecl); ok {
				lineKeys := getValidationKeys(fset, funcDecl)
				if len(lineKeys) > 0 {
					validationKeys[pkgImportPath+"."+getFuncName(funcDecl)] = lineKeys
				}
			}
		}
	}

	// Filter the struct specs to just the ones that embed rev.Controller.
	controllerSpecs := findTypesThatEmbed("rev.Controller", structSpecs)

	// Add the method specs to them.
	for _, spec := range controllerSpecs {
		spec.MethodSpecs = methodSpecs[spec.StructName]
	}

	return &SourceInfo{
		ControllerSpecs: controllerSpecs,
		ValidationKeys:  validationKeys,
		UnitTests:       findTypesThatEmbed("rev.UnitTest", structSpecs),
		FunctionalTests: findTypesThatEmbed("rev.FunctionalTest", structSpecs),
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

func addImports(imports map[string]string, decl ast.Decl) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes
		key := fullPath
		if lastSlash := strings.LastIndex(fullPath, "/"); lastSlash != -1 {
			key = fullPath[lastSlash+1:]
		}
		imports[key] = fullPath
	}
}

// If this Decl is a struct type definition, it is summarized and added to specs.
// Else, specs is returned unchanged.
func appendStruct(specs []*TypeInfo, pkgImportPath string, pkg *ast.Package, decl ast.Decl) []*TypeInfo {
	// Filter out non-Struct type declarations.
	spec, found := getStructTypeDecl(decl)
	if !found {
		return specs
	}
	structType := spec.Type.(*ast.StructType)

	// At this point we know it's a type declaration for a struct.
	// Fill in the rest of the info by diving into the fields.
	// Add it provisionally to the Controller list -- it's later filtered using field info.
	controllerSpec := &TypeInfo{
		PackageName: pkg.Name,
		StructName:  spec.Name.Name,
		ImportPath:  pkgImportPath,
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
				return pkg.Name, ident.Name
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

		controllerSpec.embeddedTypes = append(controllerSpec.embeddedTypes, &embeddedTypeName{
			PackageName: pkgName,
			StructName:  typeName,
		})
	}

	return append(specs, controllerSpec)
}

// If decl is a Method declaration, it is summarized and added to the array
// underneath its receiver type.
// e.g. "Login" => {MethodSpec, MethodSpec, ..}
func appendAction(fset *token.FileSet, mm methodMap, decl ast.Decl, pkgName string, imports map[string]string) {
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

	// Does it return a rev.Result?
	if funcDecl.Type.Results == nil || len(funcDecl.Type.Results.List) != 1 {
		return
	}
	selExpr, ok := funcDecl.Type.Results.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if pkgIdent, ok := selExpr.X.(*ast.Ident); !ok || pkgIdent.Name != "rev" {
		return
	}
	if selExpr.Sel.Name != "Result" {
		return
	}

	method := &MethodSpec{
		Name: funcDecl.Name.Name,
	}

	// Add a description of the arguments to the method.
	for _, field := range funcDecl.Type.Params.List {
		for _, name := range field.Names {
			typeName := ExprName(field.Type)

			// Figure out the Import Path for this field, if any.
			importPath := ""
			baseTypeName := strings.TrimLeft(typeName, "*")
			dotIndex := strings.Index(baseTypeName, ".")
			isExported := unicode.IsUpper([]rune(baseTypeName)[0])
			if dotIndex == -1 && isExported {
				// Fully-qualify types defined in that package.
				// (Need to add back the stars that we trimmed, too)
				typeName = pkgName + "." + baseTypeName
			} else if dotIndex != -1 {
				// The type comes from an imported package.
				argPkgName := baseTypeName[:dotIndex]
				if importPath, ok = imports[argPkgName]; !ok {
					log.Println("Failed to find import for arg of type:", typeName)
				}
			}

			method.Args = append(method.Args, &MethodArg{
				Name:       name.Name,
				TypeName:   typeName,
				ImportPath: importPath,
			})
		}
	}

	// Add a description of the calls to Render from the method.
	// Inspect every node (e.g. always return true).
	method.RenderCalls = []*methodCall{}
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
		pos := fset.Position(callExpr.Rparen)
		methodCall := &methodCall{
			Line:  pos.Line,
			Names: []string{},
		}
		for _, arg := range callExpr.Args {
			argIdent, ok := arg.(*ast.Ident)
			if !ok {
				log.Println("Unnamed argument to Render call:", pos)
				continue
			}
			methodCall.Names = append(methodCall.Names, argIdent.Name)
		}
		method.RenderCalls = append(method.RenderCalls, methodCall)
		return true
	})

	var recvTypeName string
	var recvType ast.Expr = funcDecl.Recv.List[0].Type
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
func getValidationKeys(fset *token.FileSet, funcDecl *ast.FuncDecl) map[int]string {
	var (
		lineKeys     = make(map[int]string)
		lastExprLine = 0

		// Check the func parameters and the receiver's members for the *rev.Validation type.
		validationParam = getValidationParameter(funcDecl)
	)

	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		if expr, ok := node.(*ast.ExprStmt); ok {
			lastExprLine = fset.Position(expr.End()).Line
			return true
		}

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

		// If the argument is a binary expression, take the first expression.
		// (e.g. c.Validation.Required(myName != ""))
		arg := callExpr.Args[0]
		if binExpr, ok := arg.(*ast.BinaryExpr); ok {
			arg = binExpr.X
		}

		// If it's a literal, skip it.
		if _, ok = arg.(*ast.BasicLit); ok {
			return true
		}

		lineKeys[lastExprLine] = ExprName(arg)
		return true
	})

	return lineKeys
}

// Check to see if there is a *rev.Validation as an argument.
func getValidationParameter(funcDecl *ast.FuncDecl) *ast.Object {
	for _, field := range funcDecl.Type.Params.List {
		starExpr, ok := field.Type.(*ast.StarExpr) // e.g. *rev.Validation
		if !ok {
			continue
		}

		selExpr, ok := starExpr.X.(*ast.SelectorExpr) // e.g. rev.Validation
		if !ok {
			continue
		}

		xIdent, ok := selExpr.X.(*ast.Ident) // e.g. rev
		if !ok {
			continue
		}

		if xIdent.Name == "rev" && selExpr.Sel.Name == "Validation" {
			return field.Names[0].Obj
		}
	}
	return nil
}

func (s *TypeInfo) SimpleName() string {
	return s.PackageName + "." + s.StructName
}

func (s *embeddedTypeName) SimpleName() string {
	return s.PackageName + "." + s.StructName
}

// getStructTypeDecl checks if the given decl is a type declaration for a
// struct.  If so, the TypeSpec is returned.
func getStructTypeDecl(decl ast.Decl) (spec *ast.TypeSpec, found bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.TYPE {
		return
	}

	if len(genDecl.Specs) != 1 {
		rev.TRACE.Printf("Surprising: Decl does not have 1 Spec: %v", genDecl)
		return
	}

	spec = genDecl.Specs[0].(*ast.TypeSpec)
	if _, ok := spec.Type.(*ast.StructType); ok {
		found = true
	}

	return
}

// Returnall types that (directly or indirectly) embed the target type.
func findTypesThatEmbed(targetType string, specs []*TypeInfo) (filtered []*TypeInfo) {
	// Do a search in the "embedded type graph", starting with the target type.
	nodeQueue := []string{targetType}
	for len(nodeQueue) > 0 {
		controllerSimpleName := nodeQueue[0]
		nodeQueue = nodeQueue[1:]
		for _, spec := range specs {
			if rev.ContainsString(nodeQueue, spec.SimpleName()) {
				continue // Already added
			}

			// Look through the embedded types to see if the current type is among them.
			for _, embeddedType := range spec.embeddedTypes {

				// If so, add this type's simple name to the nodeQueue, and its spec to
				// the filtered list.
				if controllerSimpleName == embeddedType.SimpleName() {
					nodeQueue = append(nodeQueue, spec.SimpleName())
					filtered = append(filtered, spec)
					break
				}
			}
		}
	}
	return
}

// This returns the syntactic expression for referencing this type in Go.
// One complexity is that package-local types have to be fully-qualified.
// For example, if the type is "Hello", then it really means "pkg.Hello".
func ExprName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return ExprName(t.X) + "." + ExprName(t.Sel)
	case *ast.StarExpr:
		return "*" + ExprName(t.X)
	case *ast.ArrayType:
		return "[]" + ExprName(t.Elt)
	default:
		log.Println("Failed to generate name for field.")
		ast.Print(nil, expr)
	}
	return ""
}
