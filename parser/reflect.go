// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package parser

// This file handles the app code introspection.
// It catalogs the controllers, their methods, and their arguments.

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

// A container used to support the reflection package.
type processContainer struct {
	root, rootImportPath string                // The paths
	paths                *model.RevelContainer // The Revel paths
	srcInfo              *model.SourceInfo     // The source information container
}

// Maps a controller simple name (e.g. "Login") to the methods for which it is a
// receiver.
type methodMap map[string][]*model.MethodSpec

// ProcessSource parses the app controllers directory and
// returns a list of the controller types found.
// Otherwise CompileError if the parsing fails.
func ProcessSource(paths *model.RevelContainer) (_ *model.SourceInfo, compileError error) {
	pc := &processContainer{paths: paths}
	for _, root := range paths.CodePaths {
		rootImportPath := importPathFromPath(root, paths.BasePath)
		if rootImportPath == "" {
			utils.Logger.Info("Skipping empty code path", "path", root)
			continue
		}
		pc.root, pc.rootImportPath = root, rootImportPath

		// Start walking the directory tree.
		compileError = utils.Walk(root, pc.processPath)
		if compileError != nil {
			return
		}
	}

	return pc.srcInfo, compileError
}

// Called during the "walk process".
func (pc *processContainer) processPath(path string, info os.FileInfo, err error) error {
	if err != nil {
		utils.Logger.Error("Error scanning app source:", "error", err)
		return nil
	}

	if !info.IsDir() || info.Name() == "tmp" {
		return nil
	}

	// Get the import path of the package.
	pkgImportPath := pc.rootImportPath
	if pc.root != path {
		pkgImportPath = pc.rootImportPath + "/" + filepath.ToSlash(path[len(pc.root)+1:])
	}

	// Parse files within the path.
	var pkgs map[string]*ast.Package
	fset := token.NewFileSet()
	pkgs, err = parser.ParseDir(
		fset,
		path,
		func(f os.FileInfo) bool {
			return !f.IsDir() && !strings.HasPrefix(f.Name(), ".") && strings.HasSuffix(f.Name(), ".go")
		},
		0)

	if err != nil {
		var errList scanner.ErrorList
		if errors.As(err, &errList) {
			pos := errList[0].Pos
			newError := &utils.SourceError{
				SourceType:  ".go source",
				Title:       "Go Compilation Error",
				Path:        pos.Filename,
				Description: errList[0].Msg,
				Line:        pos.Line,
				Column:      pos.Column,
				SourceLines: utils.MustReadLines(pos.Filename),
			}

			errorLink := pc.paths.Config.StringDefault("error.link", "")
			if errorLink != "" {
				newError.SetLink(errorLink)
			}
			return newError
		}

		// This is exception, err already checked above. Here just a print
		ast.Print(nil, err)
		utils.Logger.Fatal("Failed to parse dir", "error", err)
	}

	// Skip "main" packages.
	delete(pkgs, "main")

	// Ignore packages that end with _test
	// These cannot be included in source code that is not generated specifically as a test
	for i := range pkgs {
		if len(i) > 6 {
			if i[len(i)-5:] == "_test" {
				delete(pkgs, i)
			}
		}
	}

	// If there is no code in this directory, skip it.
	if len(pkgs) == 0 {
		return nil
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

	if pkg != nil {
		pc.srcInfo = appendSourceInfo(pc.srcInfo, processPackage(fset, pkgImportPath, path, pkg))
	} else {
		utils.Logger.Info("Ignoring package, because it contained no packages", "path", path)
	}
	return nil
}

// Process a single package within a file.
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
