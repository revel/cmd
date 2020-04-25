package parser2

import (
	//"go/ast"
	//"go/token"

	"github.com/revel/cmd/model"
	"golang.org/x/tools/go/packages"
	"github.com/revel/cmd/utils"
	"errors"

)
func ProcessSource(revelContainer *model.RevelContainer) (sourceInfo *model.SourceInfo, compileError error) {
	utils.Logger.Info("ProcessSource")
	// Combine packages for modules and app and revel
	allPackages := []string{revelContainer.ImportPath+"/app/controllers/...",model.RevelImportPath}
	for _,module := range revelContainer.ModulePathMap {
		allPackages = append(allPackages,module.ImportPath+"/app/controllers/...")
	}

	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.LoadTypes | packages.NeedTypes | packages.NeedSyntax , //|  packages.NeedImports |
			// packages.NeedTypes, // packages.LoadTypes | packages.NeedSyntax | packages.NeedTypesInfo,
			//packages.LoadSyntax | packages.NeedDeps,
		Dir:revelContainer.AppPath,
	}
	utils.Logger.Info("Before ","apppath", config.Dir,"paths",allPackages)
	pkgs, err := packages.Load(config,  allPackages...)
	utils.Logger.Info("***Loaded packegs ", "len results", len(pkgs), "error",err)
	// Lets see if we can output all the path names
	//packages.Visit(pkgs,func(p *packages.Package) bool{
	//	println("Got pre",p.ID)
	//	return true
	//}, func(p *packages.Package)  {
	//})
	counter := 0
	for _, p := range pkgs {
		utils.Logger.Info("Errores","error",p.Errors, "id",p.ID)
		//for _,g := range p.GoFiles {
		//	println("File", g)
		//}
		//for _, t:= range p.Syntax {
		//	utils.Logger.Info("File","name",t.Name)
		//}
		println("package typoe fouhnd ",p.Types.Name())
		//imports := map[string]string{}

		for _,s := range p.Syntax {
			println("File ",s.Name.Name )
			//for _, decl := range s.Decls {
			//	if decl.Tok == token.IMPORT {
			//	}
			}
		}
		//p.Fset.Iterate(func(file *token.File) bool{
		//
		//	// utils.Logger.Info("Output","Found file", p.ID," AND NAME ", f.Name())
		//	// For each declaration in the source file...
		//	//for _, decl := range file.Decls {
		//	//	addImports(imports, decl, pkgPath)
		//	//}
		//	counter ++
		//	return true
		//})
	//}

compileError = errors.New("Incompleted")
	println("*******************", counter)
	utils.Logger.Panic("Not implemented")
	return
}



// Add imports to the map from the source dir
//func addImports(imports map[string]string, decl ast.Decl, srcDir string) {
//	genDecl, ok := decl.(*ast.GenDecl)
//	if !ok {
//		return
//	}
//
//	if genDecl.Tok != token.IMPORT {
//		return
//	}
//
//	for _, spec := range genDecl.Specs {
//		importSpec := spec.(*ast.ImportSpec)
//		var pkgAlias string
//		if importSpec.Name != nil {
//			pkgAlias = importSpec.Name.Name
//			if pkgAlias == "_" {
//				continue
//			}
//		}
//		quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
//		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes
//
//		// If the package was not aliased (common case), we have to import it
//		// to see what the package name is.
//		// TODO: Can improve performance here a lot:
//		// 1. Do not import everything over and over again.  Keep a cache.
//		// 2. Exempt the standard library; their directories always match the package name.
//		// 3. Can use build.FindOnly and then use parser.ParseDir with mode PackageClauseOnly
//		if pkgAlias == "" {
//
//			utils.Logger.Debug("Reading from build", "path", fullPath, "srcPath", srcDir, "gopath", build.Default.GOPATH)
//			pkg, err := build.Import(fullPath, srcDir, 0)
//			if err != nil {
//				// We expect this to happen for apps using reverse routing (since we
//				// have not yet generated the routes).  Don't log that.
//				if !strings.HasSuffix(fullPath, "/app/routes") {
//					utils.Logger.Warn("Could not find import:", "path", fullPath, "srcPath", srcDir, "error", err)
//				}
//				continue
//			} else {
//				utils.Logger.Debug("Found package in dir", "dir", pkg.Dir, "name", pkg.ImportPath)
//			}
//			pkgAlias = pkg.Name
//		}
//
//		imports[pkgAlias] = fullPath
//	}
//}