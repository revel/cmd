package parser2

import (
	"go/ast"
	"go/token"

	"github.com/revel/cmd/model"
	"golang.org/x/tools/go/packages"
	"github.com/revel/cmd/utils"
	"errors"

	"fmt"
	"strings"
	"github.com/revel/cmd/logger"
)

type (
	SourceProcessor struct {
		revelContainer      *model.RevelContainer
		log                 logger.MultiLogger
		packageList         []*packages.Package
		importMap           map[string]string
		sourceInfoProcessor *SourceInfoProcessor
		sourceInfo          *model.SourceInfo
	}
)

func ProcessSource(revelContainer *model.RevelContainer) (sourceInfo *model.SourceInfo, compileError error) {
	utils.Logger.Info("ProcessSource")
	processor := NewSourceProcessor(revelContainer)
	compileError = processor.parse()
	sourceInfo = processor.sourceInfo
	fmt.Printf("From parsers \n%v\n%v\n", sourceInfo, compileError)
	//// Combine packages for modules and app and revel
	//allPackages := []string{revelContainer.ImportPath+"/app/controllers/...",model.RevelImportPath}
	//for _,module := range revelContainer.ModulePathMap {
	//	allPackages = append(allPackages,module.ImportPath+"/app/controllers/...")
	//}
	//allPackages = []string{revelContainer.ImportPath+"/app/controllers/..."}
	//
	//config := &packages.Config{
	//	// ode: packages.NeedSyntax | packages.NeedCompiledGoFiles,
	//	Mode: packages.NeedTypes | packages.NeedSyntax  ,
	//	//Mode:	packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
	//	//	packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile |
	//	//	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
	//	//	packages.NeedTypesSizes,
	//
	//	//Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile | packages.NeedFiles |
	//	//	packages.NeedCompiledGoFiles | packages.NeedTypesSizes |
	//	//	packages.NeedSyntax | packages.NeedCompiledGoFiles ,
	//	//Mode:  packages.NeedSyntax | packages.NeedCompiledGoFiles |  packages.NeedName | packages.NeedFiles |
	//	//	packages.LoadTypes | packages.NeedTypes | packages.NeedDeps,  //, // |
	//		// packages.NeedTypes, // packages.LoadTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	//		//packages.LoadSyntax | packages.NeedDeps,
	//	Dir:revelContainer.AppPath,
	//}
	//utils.Logger.Info("Before ","apppath", config.Dir,"paths",allPackages)
	//pkgs, err := packages.Load(config,  allPackages...)
	//utils.Logger.Info("***Loaded packegs ", "len results", len(pkgs), "error",err)
	//// Lets see if we can output all the path names
	////packages.Visit(pkgs,func(p *packages.Package) bool{
	////	println("Got pre",p.ID)
	////	return true
	////}, func(p *packages.Package)  {
	////})
	//counter := 0
	//for _, p := range pkgs {
	//	utils.Logger.Info("Errores","error",p.Errors, "id",p.ID)
	//	//for _,g := range p.GoFiles {
	//	//	println("File", g)
	//	//}
	//	//for _, t:= range p.Syntax {
	//	//	utils.Logger.Info("File","name",t.Name)
	//	//}
	//	//println("package typoe fouhnd ",p.Types.Name())
	//	//imports := map[string]string{}
	//
	//	for _,s := range p.Syntax {
	//		println("File ",s.Name.Name )
	//		for _, decl := range s.Decls {
	//			genDecl, ok := decl.(*ast.GenDecl)
	//			if !ok {
	//				continue
	//			}
	//
	//			if genDecl.Tok == token.IMPORT {
	//				for _, spec := range genDecl.Specs {
	//					importSpec := spec.(*ast.ImportSpec)
	//					fmt.Printf("*** import specification %#v\n", importSpec)
	//					var pkgAlias string
	//					if importSpec.Name != nil {
	//						pkgAlias = importSpec.Name.Name
	//						if pkgAlias == "_" {
	//							continue
	//						}
	//					}
	//					quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
	//					fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes
	//					if pkgAlias == "" {
	//						pkgAlias = fullPath
	//						if index:=strings.LastIndex(pkgAlias,"/");index>0 {
	//							pkgAlias = pkgAlias[index+1:]
	//						}
	//					}
	//					//imports[pkgAlias] = fullPath
	//					println("Package ", pkgAlias, "fullpath", fullPath)
	//				}
	//			}
	//		 }
	//		}
	//	}
	//	//p.Fset.Iterate(func(file *token.File) bool{
	//	//
	//	//	// utils.Logger.Info("Output","Found file", p.ID," AND NAME ", f.Name())
	//	//	// For each declaration in the source file...
	//	//	//for _, decl := range file.Decls {
	//	//	//	addImports(imports, decl, pkgPath)
	//	//	//}
	//	//	counter ++
	//	//	return true
	//	//})
	////}
	if false {
		compileError = errors.New("Incompleted")
		utils.Logger.Panic("Not implemented")
	}
	return
}

func NewSourceProcessor(revelContainer *model.RevelContainer) *SourceProcessor {
	s := &SourceProcessor{revelContainer:revelContainer, log:utils.Logger.New("parser", "SourceProcessor")}
	s.sourceInfoProcessor = NewSourceInfoProcessor(s)
	return s
}
func (s *SourceProcessor) parse() (compileError error) {
	if compileError = s.addPackages(); compileError != nil {
		return
	}
	if compileError = s.addImportMap(); compileError != nil {
		return
	}
	if compileError = s.addSourceInfo(); compileError != nil {
		return
	}

	return
}
func (s *SourceProcessor) addPackages() (err error) {
	allPackages := []string{s.revelContainer.ImportPath + "/..."} //,model.RevelImportPath}
	for _, module := range s.revelContainer.ModulePathMap {
		allPackages = append(allPackages, module.ImportPath + "/...") // +"/app/controllers/...")
	}
	allPackages = []string{s.revelContainer.ImportPath + "/..."} //+"/app/controllers/..."}

	config := &packages.Config{
		// ode: packages.NeedSyntax | packages.NeedCompiledGoFiles,
		Mode:
		packages.NeedTypes | // For compile error
			packages.NeedDeps | // To load dependent files
			packages.NeedName | // Loads the full package name
			packages.NeedSyntax, // To load ast tree (for end points)
		//Mode:	packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
		//	packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile |
		//	packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
		//	packages.NeedTypesSizes,

		//Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedExportsFile | packages.NeedFiles |
		//	packages.NeedCompiledGoFiles | packages.NeedTypesSizes |
		//	packages.NeedSyntax | packages.NeedCompiledGoFiles ,
		//Mode:  packages.NeedSyntax | packages.NeedCompiledGoFiles |  packages.NeedName | packages.NeedFiles |
		//	packages.LoadTypes | packages.NeedTypes | packages.NeedDeps,  //, // |
		// packages.NeedTypes, // packages.LoadTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		//packages.LoadSyntax | packages.NeedDeps,
		Dir:s.revelContainer.AppPath,
	}
	s.packageList, err = packages.Load(config, allPackages...)
	s.log.Info("Loaded packages ", "len results", len(s.packageList), "error", err)
	return
}
func (s *SourceProcessor) addImportMap() (err error) {
	s.importMap = map[string]string{}
	for _, p := range s.packageList {
		if len(p.Errors) > 0 {
			// Generate a compile error
			for _, e := range p.Errors {
				if !strings.Contains(e.Msg, "fsnotify") {
					err = utils.NewCompileError("", "", e)
				}
			}
		}
		for _, tree := range p.Syntax {
			for _, decl := range tree.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}

				if genDecl.Tok == token.IMPORT {
					for _, spec := range genDecl.Specs {
						importSpec := spec.(*ast.ImportSpec)
						//fmt.Printf("*** import specification %#v\n", importSpec)
						var pkgAlias string
						if importSpec.Name != nil {
							pkgAlias = importSpec.Name.Name
							if pkgAlias == "_" {
								continue
							}
						}
						quotedPath := importSpec.Path.Value           // e.g. "\"sample/app/models\""
						fullPath := quotedPath[1 : len(quotedPath) - 1] // Remove the quotes
						if pkgAlias == "" {
							pkgAlias = fullPath
							if index := strings.LastIndex(pkgAlias, "/"); index > 0 {
								pkgAlias = pkgAlias[index + 1:]
							}
						}
						s.importMap[pkgAlias] = fullPath
					}
				}
			}
		}
	}
	return
}

func (s *SourceProcessor) addSourceInfo() (err error) {
	for _, p := range s.packageList {
		if sourceInfo := s.sourceInfoProcessor.processPackage(p); sourceInfo != nil {
			if s.sourceInfo != nil {
				s.sourceInfo.Merge(sourceInfo)
			} else {
				s.sourceInfo = sourceInfo
			}
		}
	}
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