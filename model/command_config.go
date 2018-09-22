package model

// The constants
import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"github.com/revel/cmd/utils"
)

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
		Index        COMMAND  // The index
		Verbose      bool     `short:"v" long:"debug" description:"If set the logger is set to verbose"`              // True if debug is active
		HistoricMode bool     `long:"historic-run-mode" description:"If set the runmode is passed a string not json"` // True if debug is active
		ImportPath   string   // The import path (converted from various commands)
		GoPath       string   // The GoPath
		GoCmd        string   // The full path to the go executable
		SrcRoot      string   // The source root
		AppPath      string   // The application path
		AppName      string   // The applicaiton name
		BasePath     string   // The base path
		SkeletonPath string   // The skeleton path
		BuildFlags   []string `short:"X" long:"build-flags" description:"These flags will be used when building the application. May be specified multiple times, only applicable for Build, Run, Package, Test commands"`
		// The new command
		New struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Skeleton   string `short:"s" long:"skeleton" description:"Path to skeleton folder (Must exist on GO PATH)" required:"false"`
			Vendored   bool   `short:"V" long:"vendor" description:"True if project should contain a vendor folder to be initialized. Creates the vendor folder and the 'Gopkg.toml' file in the root"`
			Run        bool   `short:"r" long:"run" description:"True if you want to run the application right away"`
		} `command:"new"`
		// The build command
		Build struct {
			TargetPath string `short:"t" long:"target-path" description:"Path to target folder. Folder will be completely deleted if it exists" required:"true"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder"  required:"true"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
		} `command:"build"`
		// The run command
		Run struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder"  required:"true"`
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			Port       string `short:"p" long:"port" description:"The port to listen"`
			NoProxy    bool   `short:"n" long:"no-proxy" description:"True if proxy server should not be started. This will only update the main and routes files on change"`
		} `command:"run"`
		// The package command
		Package struct {
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder"  required:"true"`
			CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
		} `command:"package"`
		// The clean command
		Clean struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder"  required:"true"`
		} `command:"clean"`
		// The test command
		Test struct {
			Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"true"`
			Function   string `short:"f" long:"suite-function" description:"The suite.function"`
		} `command:"test"`
		// The version command
		Version struct {
			ImportPath string `short:"a" long:"application-path" description:"Path to applicaiton folder" required:"false"`
		} `command:"version"`
	}
)

// Updates the import path depending on the command
func (c *CommandConfig) UpdateImportPath() bool {
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
				if strings.HasPrefix(currentPath, path)  {
					importPath = currentPath[len(path) + 1:]
					// Remove the source from the path if it is there
					if len(importPath)>4 && strings.ToLower(importPath[0:4]) == "src/" {
						importPath = importPath[4:]
					} else if importPath == "src" {
						importPath = ""
					}
					utils.Logger.Info("Updated import path", "path", importPath)
				}
			}
		}
	}

	c.ImportPath = importPath
	utils.Logger.Info("Returned import path", "path", importPath, "buildpath",build.Default.GOPATH)
	return (len(importPath) > 0 || !required)
}
