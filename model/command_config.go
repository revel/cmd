package model

import (
	"fmt"
	"github.com/revel/cmd"
//	"github.com/revel/cmd/logger"
	"github.com/revel/cmd/utils"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"github.com/revel/cmd/model/command"
)

// The constants
const (
	NEW COMMAND = iota + 1
	RUN
	BUILD
	PACKAGE
	CLEAN
	TEST
	VERSION
)

type (
	// The Revel command type
	COMMAND int

	// The Command config for the line input
	CommandConfig struct {
		Index            COMMAND                    // The index
		Verbose          []bool                     `short:"v" long:"debug" description:"If set the logger is set to verbose"` // True if debug is active
		FrameworkVersion *Version                   // The framework version
		CommandVersion   *Version                   // The command version
		HistoricMode     bool                       `long:"historic-run-mode" description:"If set the runmode is passed a string not json"` // True if debug is active
		ImportPath       string                     // The import path (relative to a GOPATH)
		GoPath           string                     // The GoPath
		GoCmd            string                     // The full path to the go executable
		SrcRoot          string                     // The source root
		AppPath          string                     // The application path (absolute)
		AppName          string                     // The application name
		HistoricBuildMode bool                     `long:"historic-build-mode" description:"If set the code is scanned using the original parsers, not the go.1.11+"` // True if debug is active
		Vendored          bool                     // True if the application is vendored
		PackageResolver  func(pkgName string) error //  a packge resolver for the config
		BuildFlags       []string                   `short:"X" long:"build-flags" description:"These flags will be used when building the application. May be specified multiple times, only applicable for Build, Run, Package, Test commands"`

		New command.New `command:"new"` // The new command
		// The build command
		Build command.Build `command:"build"`
		// The run command
		Run command.Run `command:"run"`
		// The package command
		Package command.Package `command:"package"`
		// The clean command
		Clean command.Clean `command:"clean"`
		// The test command
		Test command.Test `command:"test"`
		// The version command
		Version command.Version `command:"version"`
	}
)

// Updates the import path depending on the command
func (c *CommandConfig) UpdateImportPath() error {
	var importPath string
	required := true
	switch c.Index {
	case NEW:
		importPath = c.New.ImportPath
	case RUN:
		importPath = c.Run.ImportPath
		c.Vendored = utils.Exists(filepath.Join(importPath,"go.mod"))
	case BUILD:
		importPath = c.Build.ImportPath
		c.Vendored = utils.Exists(filepath.Join(importPath,"go.mod"))
	case PACKAGE:
		importPath = c.Package.ImportPath
		c.Vendored = utils.Exists(filepath.Join(importPath,"go.mod"))
	case CLEAN:
		importPath = c.Clean.ImportPath
		c.Vendored = utils.Exists(filepath.Join(importPath,"go.mod"))
	case TEST:
		importPath = c.Test.ImportPath
		c.Vendored = utils.Exists(filepath.Join(importPath,"go.mod"))
	case VERSION:
		importPath = c.Version.ImportPath
		required = false
	}

	if len(importPath) == 0 || filepath.IsAbs(importPath) || importPath[0] == '.' {
		utils.Logger.Info("Import path is absolute or not specified", "path", importPath)
		// Try to determine the import path from the GO paths and the command line
		currentPath, err := os.Getwd()
		if len(importPath) > 0 {
			if importPath[0] == '.' {
				// For a relative path
				importPath = filepath.Join(currentPath, importPath)
			}
			// For an absolute path
			currentPath, _ = filepath.Abs(importPath)
		}

		if err == nil {
			for _, path := range strings.Split(build.Default.GOPATH, string(filepath.ListSeparator)) {
				utils.Logger.Infof("Checking import path %s with %s", currentPath, path)
				if strings.HasPrefix(currentPath, path) && len(currentPath) > len(path)+1 {
					importPath = currentPath[len(path)+1:]
					// Remove the source from the path if it is there
					if len(importPath) > 4 && strings.ToLower(importPath[0:4]) == "src/" {
						importPath = importPath[4:]
					} else if importPath == "src" {
						if c.Index != VERSION {
							return fmt.Errorf("Invlaid import path, working dir is in GOPATH root")
						}
						importPath = ""
					}
					utils.Logger.Info("Updated import path", "path", importPath)
				}
			}
		}
	}

	c.ImportPath = importPath
	// We need the source root determined at this point to check the setversions
	c.initAppFolder()
	utils.Logger.Info("Returned import path", "path", importPath)
	if required && c.Index != NEW {
		if err := c.SetVersions(); err != nil {
			utils.Logger.Panic("Failed to fetch revel versions", "error", err)
		}
		if err:=c.FrameworkVersion.CompatibleFramework(c);err!=nil {
			utils.Logger.Fatal("Compatibility Error", "message", err,
				"Revel framework version", c.FrameworkVersion.String(), "Revel tool version", c.CommandVersion.String())
		}
		utils.Logger.Info("Revel versions", "revel-tool", c.CommandVersion.String(), "Revel Framework", c.FrameworkVersion.String())
	}
	if !required {
		return nil
	}
	if len(importPath) == 0  {
		return fmt.Errorf("Unable to determine import path from : %s", importPath)
	}
	return nil
}

func (c *CommandConfig) initAppFolder() (err error) {
	utils.Logger.Info("initAppFolder","vendored", c.Vendored)

	// check for go executable
	c.GoCmd, err = exec.LookPath("go")
	if err != nil {
		utils.Logger.Fatal("Go executable not found in PATH.")
	}

	// First try to determine where the application is located - this should be the import value
	appFolder := c.ImportPath
	wd,err := os.Getwd()
	if len(appFolder) == 0 {
		// We will assume the working directory is the appFolder
		appFolder = wd
	}  else if strings.LastIndex(wd,appFolder)==len(wd)-len(appFolder) {
		// Check for existence of an /app folder
		if utils.Exists(filepath.Join(wd,"app")) {
			appFolder = wd
		} else {
			appFolder = filepath.Join(wd,appFolder)
		}
	} else if strings.Contains(appFolder,".") {
		appFolder = filepath.Join(wd,filepath.Base(c.ImportPath))
	} else if !filepath.IsAbs(appFolder) {
		appFolder = filepath.Join(wd,appFolder)
	}

	utils.Logger.Info("Determined app folder to be", "appfolder",appFolder, "working",wd,"importPath",c.ImportPath)

	// Use app folder to read the go.mod if it exists and extract the package information
	goModFile := filepath.Join(appFolder,"go.mod")
	if utils.Exists(goModFile) {
		c.Vendored = true
		file,err:=ioutil.ReadFile(goModFile)
		if err!=nil {
			return err
		}
		for _,line := range strings.Split(string(file),"\n") {
			if strings.Index(line,"module ")==0 {
				c.ImportPath = strings.TrimSpace(strings.Split(line,"module")[1])
				c.AppPath = appFolder
				c.SrcRoot = appFolder
				utils.Logger.Info("Set application path and package based on go mod", "path", c.AppPath, "sourceroot", c.SrcRoot)
				return nil
			}
		}
	}

	utils.Logger.Debug("Trying to set path based on gopath")
	// lookup go path
	c.GoPath = build.Default.GOPATH
	if c.GoPath == "" {
		utils.Logger.Fatal("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	// revel/revel#1004 choose go path relative to current working directory

	// What we want to do is to add the import to the end of the
	// gopath, and discover which import exists - If none exist this is an error except in the case
	// where we are dealing with new which is a special case where we will attempt to target the working directory first
	workingDir, _ := os.Getwd()
	goPathList := filepath.SplitList(c.GoPath)
	bestpath := ""
	if !c.Vendored {
		for _, path := range goPathList {
			if c.Index == NEW {
				// If the GOPATH is part of the working dir this is the most likely target
				if strings.HasPrefix(workingDir, path) {
					bestpath = path
				}
			} else {
				if utils.Exists(filepath.Join(path, "src", c.ImportPath)) {
					c.SrcRoot = path
					break
				}
			}
		}
		if len(c.SrcRoot) == 0 && len(bestpath) > 0 {
			c.SrcRoot = bestpath
		}

	} else {
		c.SrcRoot = appFolder
	}

	utils.Logger.Info("Source root", "path", c.SrcRoot, "cwd", workingDir, "gopath", c.GoPath, "bestpath",bestpath)

	// If source root is empty and this isn't a version then skip it
	if len(c.SrcRoot) == 0 {
		if c.Index == NEW {
			c.SrcRoot = c.New.ImportPath
		} else {
			if c.Index != VERSION {
				utils.Logger.Fatal("Abort: could not create a Revel application outside of GOPATH.")
			}
			return nil
		}
	}

	// set go src path
	if c.Vendored {
		c.AppPath = c.SrcRoot

	} else {
		c.SrcRoot = filepath.Join(c.SrcRoot, "src")

		c.AppPath = filepath.Join(c.SrcRoot, filepath.FromSlash(c.ImportPath))
	}
	utils.Logger.Info("Set application path", "path", c.AppPath)
	return nil
}

// Used to initialize the package resolver
func (c *CommandConfig) InitPackageResolver() {
	utils.Logger.Info("InitPackageResolver", "useVendor", c.Vendored, "path", c.AppPath)

	// This should get called when needed
	c.PackageResolver = func(pkgName string) error {
		//useVendor := utils.DirExists(filepath.Join(c.AppPath, "vendor"))

		//var getCmd *exec.Cmd
		utils.Logger.Info("Request for package ", "package", pkgName, "use vendor", c.Vendored)
		if c.Vendored {
			goModCmd := exec.Command("go", "mod", "tidy")
			utils.CmdInit(goModCmd,!c.Vendored, c.AppPath)
			goModCmd.Run()
			return nil
		}

		return nil
	}
}

// lookup and set Go related variables
func (c *CommandConfig) InitGoPathsOld() {
	utils.Logger.Info("InitGoPaths")
	// lookup go path
	c.GoPath = build.Default.GOPATH
	if c.GoPath == "" {
		utils.Logger.Fatal("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	// check for go executable
	var err error
	c.GoCmd, err = exec.LookPath("go")
	if err != nil {
		utils.Logger.Fatal("Go executable not found in PATH.")
	}

	// revel/revel#1004 choose go path relative to current working directory

	// What we want to do is to add the import to the end of the
	// gopath, and discover which import exists - If none exist this is an error except in the case
	// where we are dealing with new which is a special case where we will attempt to target the working directory first
	workingDir, _ := os.Getwd()
	goPathList := filepath.SplitList(c.GoPath)
	bestpath := ""
	for _, path := range goPathList {
		if c.Index == NEW {
			// If the GOPATH is part of the working dir this is the most likely target
			if strings.HasPrefix(workingDir, path) {
				bestpath = path
			}
		} else {
			if utils.Exists(filepath.Join(path, "src", c.ImportPath)) {
				c.SrcRoot = path
				break
			}
		}
	}

	utils.Logger.Info("Source root", "path", c.SrcRoot, "cwd", workingDir, "gopath", c.GoPath, "bestpath",bestpath)
	if len(c.SrcRoot) == 0 && len(bestpath) > 0 {
		c.SrcRoot = bestpath
	}

	// If source root is empty and this isn't a version then skip it
	if len(c.SrcRoot) == 0 {
		if c.Index == NEW {
			c.SrcRoot = c.New.ImportPath
		} else {
			if c.Index != VERSION {
				utils.Logger.Fatal("Abort: could not create a Revel application outside of GOPATH.")
			}
			return
		}
	}

	// set go src path
	c.SrcRoot = filepath.Join(c.SrcRoot, "src")

	c.AppPath = filepath.Join(c.SrcRoot, filepath.FromSlash(c.ImportPath))
	utils.Logger.Info("Set application path", "path", c.AppPath)
}

// Sets the versions on the command config
func (c *CommandConfig) SetVersions() (err error) {
	c.CommandVersion, _ = ParseVersion(cmd.Version)
	pathMap, err := utils.FindSrcPaths(c.AppPath, []string{RevelImportPath}, c.PackageResolver)
	if err == nil {
		utils.Logger.Info("Fullpath to revel", "dir", pathMap[RevelImportPath])
		fset := token.NewFileSet() // positions are relative to fset

		versionData, err := ioutil.ReadFile(filepath.Join(pathMap[RevelImportPath],  "version.go"))
		if err != nil {
			utils.Logger.Error("Failed to find Revel version:", "error", err, "path", pathMap[RevelImportPath])
		}

		// Parse src but stop after processing the imports.
		f, err := parser.ParseFile(fset, "", versionData, parser.ParseComments)
		if err != nil {
			return utils.NewBuildError("Failed to parse Revel version error:", "error", err)
		}

		// Print the imports from the file's AST.
		for _, s := range f.Decls {
			genDecl, ok := s.(*ast.GenDecl)
			if !ok {
				continue
			}
			if genDecl.Tok != token.CONST {
				continue
			}
			for _, a := range genDecl.Specs {
				spec := a.(*ast.ValueSpec)
				r := spec.Values[0].(*ast.BasicLit)
				if spec.Names[0].Name == "Version" {
					c.FrameworkVersion, err = ParseVersion(strings.Replace(r.Value, `"`, ``, -1))
					if err != nil {
						utils.Logger.Errorf("Failed to parse version")
					} else {
						utils.Logger.Info("Parsed revel version", "version", c.FrameworkVersion.String())
					}
				}
			}
		}
	}
	return
}
