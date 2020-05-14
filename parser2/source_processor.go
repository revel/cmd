package parser2

import (
	"github.com/revel/cmd/model"
	"golang.org/x/tools/go/packages"
	"github.com/revel/cmd/utils"
	"errors"

	"strings"
	"github.com/revel/cmd/logger"
)

type (
	SourceProcessor struct {
		revelContainer      *model.RevelContainer
		log                 logger.MultiLogger
		packageList         []*packages.Package
		importMap           map[string]string
		packageMap          map[string]string
		sourceInfoProcessor *SourceInfoProcessor
		sourceInfo          *model.SourceInfo
	}
)

func ProcessSource(revelContainer *model.RevelContainer) (sourceInfo *model.SourceInfo, compileError error) {
	utils.Logger.Info("ProcessSource")
	processor := NewSourceProcessor(revelContainer)
	compileError = processor.parse()
	sourceInfo = processor.sourceInfo
	if compileError == nil {
		processor.log.Infof("From parsers : Structures:%d InitImports:%d ValidationKeys:%d %v", len(sourceInfo.StructSpecs), len(sourceInfo.InitImportPaths), len(sourceInfo.ValidationKeys), sourceInfo.PackageMap)
	}

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
	s.sourceInfo.PackageMap = map[string]string{}
	getImportFromMap := func(packagePath string) string {
		for path := range s.packageMap {
			if strings.Index(path, packagePath) == 0 {
				fullPath := s.packageMap[path]
				return fullPath[:(len(fullPath) - len(path) + len(packagePath))]
			}
		}
		return ""
	}
	s.sourceInfo.PackageMap[model.RevelImportPath] = getImportFromMap(model.RevelImportPath)
	s.sourceInfo.PackageMap[s.revelContainer.ImportPath] = getImportFromMap(s.revelContainer.ImportPath)
	for _, module := range s.revelContainer.ModulePathMap {
		s.sourceInfo.PackageMap[module.ImportPath] = getImportFromMap(module.ImportPath)
	}

	return
}

// Using the packages.Load function load all the packages and type specifications (forces compile).
// this sets the SourceProcessor.packageList         []*packages.Package
func (s *SourceProcessor) addPackages() (err error) {
	allPackages := []string{s.revelContainer.ImportPath + "/...", model.RevelImportPath + "/..."}
	for _, module := range s.revelContainer.ModulePathMap {
		allPackages = append(allPackages, module.ImportPath + "/...") // +"/app/controllers/...")
	}
	s.log.Info("Reading packages", "packageList", allPackages)
	//allPackages = []string{s.revelContainer.ImportPath + "/..."} //+"/app/controllers/..."}

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

// This function is used to populate a map so that we can lookup controller embedded types in order to determine
// if a Struct inherits from from revel.Controller
func (s *SourceProcessor) addImportMap() (err error) {
	s.importMap = map[string]string{}
	s.packageMap = map[string]string{}
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
			s.importMap[tree.Name.Name] = p.PkgPath
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
