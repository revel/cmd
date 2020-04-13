package model

import (
	"fmt"
	"github.com/revel/cmd"
	"github.com/revel/cmd/logger"
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
		Vendored          bool                     // True if the application is vendored
		PackageResolver  func(pkgName string) error //  a packge resolver for the config
		BuildFlags       []string                   `short:"X" long:"build-flags" description:"These flags will be used when building the application. May be specified multiple times, only applicable for Build, Run, Package, Test commands"`
		// The new command
		New struct {
			ImportPath   string `short:"a" long:"application-path" description:"Path to application folder" required:"false"`
			SkeletonPath string `short:"s" long:"skeleton" description:"Path to skeleton folder (Must exist on GO PATH)" required:"false"`
			Vendored     bool   `short:"V" long:"vendor" description:"True if project should contain a vendor folder to be initialized. Creates the vendor folder and the 'Gopkg.toml' file in the root"`
			Run          bool   `short:"r" long:"run" description:"True if you want to run the application right away"`
		} `command:"new"`
		// The build command
		Build struct {
			TargetPath string `short:"t" long:"target-path" description:"Path to target folder. Folder will be completely deleted if it exists" required:"false"`
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder"  required:"false"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
		} `command:"build"`
		// The run command
		Run struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder"  required:"false"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			Port       int    `short:"p" long:"port" default:"-1" description:"The port to listen" `
			NoProxy    bool   `short:"n" long:"no-proxy" description:"True if proxy server should not be started. This will only update the main and routes files on change"`
		} `command:"run"`
		// The package command
		Package struct {
			TargetPath string `short:"t" long:"target-path" description:"Full path and filename of target package to deploy" required:"false"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder"  required:"false"`
			CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
		} `command:"package"`
		// The clean command
		Clean struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder"  required:"false"`
		} `command:"clean"`
		// The test command
		Test struct {
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder" required:"false"`
			Function   string `short:"f" long:"suite-function" description:"The suite.function"`
		} `command:"test"`
		// The version command
		Version struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to application folder" required:"false"`
			Update bool `short:"u" long:"update" description:"Update the framework and modules" required:"false"`
		} `command:"version"`
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
	case BUILD:
		importPath = c.Build.ImportPath
	case PACKAGE:
		importPath = c.Package.ImportPath
	case CLEAN:
		importPath = c.Clean.ImportPath
	case TEST:
		importPath = c.Test.ImportPath
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
	utils.Logger.Info("Returned import path", "path", importPath, "buildpath", build.Default.GOPATH)
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

// Used to initialize the package resolver
func (c *CommandConfig) InitPackageResolver() {
	c.Vendored = utils.DirExists(filepath.Join(c.AppPath, "vendor"))
	if c.Index == NEW && c.New.Vendored {
		c.Vendored = true
	}

	utils.Logger.Info("InitPackageResolver", "useVendor", c.Vendored, "path", c.AppPath)

	var (
		depPath string
		err     error
	)

	if c.Vendored {
		utils.Logger.Info("Vendor folder detected, scanning for deps in path")
		depPath, err = exec.LookPath("dep")
		if err != nil {
			// Do not halt build unless a new package needs to be imported
			utils.Logger.Fatal("Build: `dep` executable not found in PATH, but vendor folder detected." +
				"Packages can only be added automatically to the vendor folder using the `dep` tool. " +
				"You can install the `dep` tool by doing a `go get -u github.com/golang/dep/cmd/dep`")
		}
	}

	// This should get called when needed
	c.PackageResolver = func(pkgName string) error {
		//useVendor := utils.DirExists(filepath.Join(c.AppPath, "vendor"))

		var getCmd *exec.Cmd
		utils.Logger.Info("Request for package ", "package", pkgName, "use vendor", c.Vendored)
		if c.Vendored {
			utils.Logger.Info("Using dependency manager to import package", "package", pkgName)

			if depPath == "" {
				utils.Logger.Error("Build: Vendor folder found, but the `dep` tool was not found, " +
					"if you use a different vendoring (package management) tool please add the following packages by hand, " +
					"or install the `dep` tool into your gopath by doing a `go get -u github.com/golang/dep/cmd/dep`. " +
					"For more information and usage of the tool please see http://github.com/golang/dep")
				utils.Logger.Error("Missing package", "package", pkgName)
				return fmt.Errorf("Missing package %s", pkgName)
			}
			// Check to see if the package exists locally
			_, err := build.Import(pkgName, c.AppPath, build.FindOnly)
			if err != nil {
				getCmd = exec.Command(depPath, "ensure", "-add", pkgName)
			} else {
				getCmd = exec.Command(depPath, "ensure", "-update", pkgName)
			}


		} else {
			utils.Logger.Info("No vendor folder detected, not using dependency manager to import package", "package", pkgName)
			getCmd = exec.Command(c.GoCmd, "get", "-u", pkgName)
		}

		utils.CmdInit(getCmd, c.AppPath)
		utils.Logger.Info("Go get command ", "exec", getCmd.Path, "dir", getCmd.Dir, "args", getCmd.Args, "env", getCmd.Env, "package", pkgName)
		output, err := getCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Error stack %v\n", logger.NewCallStack())
			utils.Logger.Error("Failed to import package", "error", err, "gopath", build.Default.GOPATH, "GO-ROOT", build.Default.GOROOT, "output", string(output))
		}
		return err
	}
}

// lookup and set Go related variables
func (c *CommandConfig) InitGoPaths() {
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
		if c.Index != VERSION {
			utils.Logger.Fatal("Abort: could not create a Revel application outside of GOPATH.")
		}
		return
	}

	// set go src path
	c.SrcRoot = filepath.Join(c.SrcRoot, "src")

	c.AppPath = filepath.Join(c.SrcRoot, filepath.FromSlash(c.ImportPath))
	utils.Logger.Info("Set application path", "path", c.AppPath)
}

// Sets the versions on the command config
func (c *CommandConfig) SetVersions() (err error) {
	c.CommandVersion, _ = ParseVersion(cmd.Version)
	_, revelPath, err := utils.FindSrcPaths(c.ImportPath, RevelImportPath, c.PackageResolver)
	if err == nil {
		utils.Logger.Info("Fullpath to revel", "dir", revelPath)
		fset := token.NewFileSet() // positions are relative to fset

		versionData, err := ioutil.ReadFile(filepath.Join(revelPath, RevelImportPath, "version.go"))
		if err != nil {
			utils.Logger.Error("Failed to find Revel version:", "error", err, "path", revelPath)
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
