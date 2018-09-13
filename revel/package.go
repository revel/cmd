// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

var cmdPackage = &Command{
	UsageLine: "package -i [import path] -r [run mode]",
	Short:     "package a Revel application (e.g. for deployment)",
	Long: `
Package the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

For example:

    revel package -i github.com/revel/examples/chat
`,
}

func init() {
	cmdPackage.RunWith = packageApp
	cmdPackage.UpdateConfig = updatePackageConfig
}

// Called when unable to parse the command line automatically and assumes an old launch
func updatePackageConfig(c *model.CommandConfig, args []string) bool {
	c.Index = PACAKAGE
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, cmdPackage.Long)
		return false
	}
	c.New.ImportPath = args[0]
	if len(args)>1 {
		c.New.Skeleton = args[1]
	}
	return true

}

func packageApp(c *model.CommandConfig) {

	// Determine the run mode.
	mode := DefaultRunMode
	if len(c.Package.Mode) >= 0 {
		mode = c.Package.Mode
	}

	appImportPath := c.Package.ImportPath
	revel_paths := model.NewRevelPaths(mode, appImportPath, "", model.DoNothingRevelCallback)

	// Remove the archive if it already exists.
	destFile := filepath.Base(revel_paths.BasePath) + ".tar.gz"
	if err := os.Remove(destFile); err != nil && !os.IsNotExist(err) {
		utils.Logger.Error("Unable to remove target file","error",err,"file",destFile)
		os.Exit(1)
	}

	// Collect stuff in a temp directory.
	tmpDir, err := ioutil.TempDir("", filepath.Base(revel_paths.BasePath))
	utils.PanicOnError(err, "Failed to get temp dir")

	// Build expects the command the build to contain the proper data
	c.Build.ImportPath = appImportPath
	c.Build.Mode = mode
	c.Build.TargetPath = tmpDir
	buildApp(c)

	// Create the zip file.
	archiveName := utils.MustTarGzDir(destFile, tmpDir)

	fmt.Println("Your archive is ready:", archiveName)
}
