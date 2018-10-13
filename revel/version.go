// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"github.com/revel/cmd"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

var cmdVersion = &Command{
	UsageLine: "revel version",
	Short:     "displays the Revel Framework and Go version",
	Long: `
Displays the Revel Framework and Go version.

For example:

    revel version [<application path>]
`,
}

func init() {
	cmdVersion.RunWith = versionApp
	cmdVersion.UpdateConfig = updateVersionConfig
}

// Update the version
func updateVersionConfig(c *model.CommandConfig, args []string) bool {
	if len(args) > 0 {
		c.Version.ImportPath = args[0]
	}
	return true
}

// Displays the version of go and Revel
func versionApp(c *model.CommandConfig) (err error) {

	var revelPath, appPath string


	appPath, revelPath, err = utils.FindSrcPaths(c.ImportPath, model.RevelImportPath, c.PackageResolver)
	if err != nil {
		return utils.NewBuildError("Failed to import "+c.ImportPath+" with error:", "error", err)
	}
	revelPath = revelPath + model.RevelImportPath

	fmt.Println("\nRevel Framework",revelPath, appPath )
	if err != nil {
		utils.Logger.Info("Failed to find Revel in GOPATH with error:", "error", err, "gopath", build.Default.GOPATH)
		fmt.Println("Information not available (not on GOPATH)")
	} else {
		utils.Logger.Info("Fullpath to revel", "dir", revelPath)
		fset := token.NewFileSet() // positions are relative to fset

		version, err := ioutil.ReadFile(filepath.Join(revelPath, "version.go"))
		if err != nil {
			utils.Logger.Error("Failed to find Revel version:", "error", err, "path", revelPath)
		}

		// Parse src but stop after processing the imports.
		f, err := parser.ParseFile(fset, "", version, parser.ParseComments)
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
				fmt.Printf("Revel %s = %s\n", spec.Names[0].Name, r.Value)
			}
		}
	}
	fmt.Println("\nRevel Command Utility Tool")
	fmt.Println("Version", cmd.Version)
	fmt.Println("Build Date", cmd.BuildDate)
	fmt.Println("Minimum Go Version", cmd.MinimumGoVersion)

	fmt.Printf("Compiled By   %s %s/%s\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	// Extract the goversion detected
	if len(c.GoCmd) > 0 {
		cmd := exec.Command(c.GoCmd, "version")
		cmd.Stdout = os.Stdout
		if e := cmd.Start(); e != nil {
			fmt.Println("Go command error ", e)
		} else {
			cmd.Wait()
		}
	} else {
		fmt.Println("Go command not found ")
	}

	return
}

