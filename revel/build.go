// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/harness"
	"github.com/revel/cmd/model"
	"go/build"
	"github.com/revel/cmd/utils"
	"fmt"
)

var cmdBuild = &Command{
	UsageLine: "build -i [import path] -t [target path] -r [run mode]",
	Short:     "build a Revel application (e.g. for deployment)",
	Long: `
Build the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

For example:

    revel build -a github.com/revel/examples/chat -t /tmp/chat

`,
}

func init() {
	cmdBuild.RunWith = buildApp
	cmdBuild.UpdateConfig = updateBuildConfig
}

// The update config updates the configuration command so that it can run
func updateBuildConfig(c *model.CommandConfig, args []string) (bool) {
	c.Index = BUILD
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
		return false
	}
	c.Build.ImportPath = args[0]
	c.Build.TargetPath = args[1]
	if len(args)>2 {
		c.Build.Mode = args[1]
	}
	return true
}

// The main entry point to build application from command line
func buildApp(c *model.CommandConfig) {
	c.ImportPath = c.Build.ImportPath
	appImportPath, destPath, mode := c.Build.ImportPath , c.Build.TargetPath, DefaultRunMode
	if len(c.Build.Mode) > 0 {
		mode = c.Build.Mode
	}

	// Convert target to absolute path
	destPath, _ = filepath.Abs(destPath)

	revel_paths := model.NewRevelPaths(mode, appImportPath, "", model.DoNothingRevelCallback)

	// First, verify that it is either already empty or looks like a previous
	// build (to avoid clobbering anything)
	if utils.Exists(destPath) && !utils.Empty(destPath) && !utils.Exists(filepath.Join(destPath, "run.sh")) {
		utils.Logger.Errorf("Abort: %s exists and does not look like a build directory.", destPath)
		return
	}

	if err := os.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
		utils.Logger.Error("Remove all error","error", err)
		return
	}

	if err := os.MkdirAll(destPath, 0777); err != nil {
		utils.Logger.Error("makedir error","error",err)
		return
	}

	app, reverr := harness.Build(c,revel_paths)
	if reverr!=nil {
		utils.Logger.Error("Failed to build application","error",reverr)
		return
	}

	// Included are:
	// - run scripts
	// - binary
	// - revel
	// - app

	// Revel and the app are in a directory structure mirroring import path
	srcPath := filepath.Join(destPath, "src")
	destBinaryPath := filepath.Join(destPath, filepath.Base(app.BinaryPath))
	tmpRevelPath := filepath.Join(srcPath, filepath.FromSlash(model.RevelImportPath))
	utils.MustCopyFile(destBinaryPath, app.BinaryPath)
	utils.MustChmod(destBinaryPath, 0755)
	_ = utils.MustCopyDir(filepath.Join(tmpRevelPath, "conf"), filepath.Join(revel_paths.RevelPath, "conf"), nil)
	_ = utils.MustCopyDir(filepath.Join(tmpRevelPath, "templates"), filepath.Join(revel_paths.RevelPath, "templates"), nil)
	_ = utils.MustCopyDir(filepath.Join(srcPath, filepath.FromSlash(appImportPath)), revel_paths.BasePath, nil)

	// Find all the modules used and copy them over.
	config := revel_paths.Config.Raw()
	modulePaths := make(map[string]string) // import path => filesystem path
	for _, section := range config.Sections() {
		options, _ := config.SectionOptions(section)
		for _, key := range options {
			if !strings.HasPrefix(key, "module.") {
				continue
			}
			moduleImportPath, _ := config.String(section, key)
			if moduleImportPath == "" {
				continue
			}

			modPkg, err := build.Import(c.ImportPath, revel_paths.RevelPath, build.FindOnly)
			if err != nil {
				utils.Logger.Fatalf("Failed to load module %s (%s): %s", key[len("module."):], c.ImportPath, err)
			}
			modulePaths[moduleImportPath] = modPkg.Dir
		}
	}
	for importPath, fsPath := range modulePaths {
		_ = utils.MustCopyDir(filepath.Join(srcPath, importPath), fsPath, nil)
	}

	tmplData := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
		"Mode":       mode,
	}

	utils.MustGenerateTemplate(
		filepath.Join(destPath, "run.sh"),
		PACKAGE_RUN_SH,
		tmplData,
	)
	utils.MustChmod(filepath.Join(destPath, "run.sh"), 0755)
	utils.MustGenerateTemplate(
		filepath.Join(destPath, "run.bat"),
		PACKAGE_RUN_BAT,
		tmplData,
	)

	fmt.Println("Your application has been built in:", destPath)

}

const PACKAGE_RUN_SH = `#!/bin/sh

SCRIPTPATH=$(cd "$(dirname "$0")"; pwd)
"$SCRIPTPATH/{{.BinName}}" -importPath {{.ImportPath}} -srcPath "$SCRIPTPATH/src" -runMode {{.Mode}}
`
const PACKAGE_RUN_BAT = `@echo off

{{.BinName}} -importPath {{.ImportPath}} -srcPath "%CD%\src" -runMode {{.Mode}}
`
