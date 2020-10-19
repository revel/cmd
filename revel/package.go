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
	UsageLine: "package [-r [run mode]] [application] ",
	Short:     "package a Revel application (e.g. for deployment)",
	Long: `
Package the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

For example:

    revel package github.com/revel/examples/chat
`,
}

func init() {
	cmdPackage.RunWith = packageApp
	cmdPackage.UpdateConfig = updatePackageConfig
}

// Called when unable to parse the command line automatically and assumes an old launch.
func updatePackageConfig(c *model.CommandConfig, args []string) bool {
	c.Index = model.PACKAGE
	if len(args) == 0 && c.Package.ImportPath != "" {
		return true
	}
	c.Package.ImportPath = args[0]
	if len(args) > 1 {
		c.Package.Mode = args[1]
	}
	return true
}

// Called to package the app.
func packageApp(c *model.CommandConfig) (err error) {
	// Determine the run mode.
	mode := c.Package.Mode

	appImportPath := c.ImportPath
	revelPaths, err := model.NewRevelPaths(mode, appImportPath, c.AppPath, model.NewWrappedRevelCallback(nil, c.PackageResolver))
	if err != nil {
		return
	}

	// Remove the archive if it already exists.
	destFile := filepath.Join(c.AppPath, filepath.Base(revelPaths.BasePath)+".tar.gz")
	if c.Package.TargetPath != "" {
		if filepath.IsAbs(c.Package.TargetPath) {
			destFile = c.Package.TargetPath
		} else {
			destFile = filepath.Join(c.AppPath, c.Package.TargetPath)
		}
	}
	if err := os.Remove(destFile); err != nil && !os.IsNotExist(err) {
		return utils.NewBuildError("Unable to remove target file", "error", err, "file", destFile)
	}

	// Collect stuff in a temp directory.
	tmpDir, err := ioutil.TempDir("", filepath.Base(revelPaths.BasePath))
	utils.PanicOnError(err, "Failed to get temp dir")

	// Build expects the command the build to contain the proper data
	c.Build.Mode = c.Package.Mode
	c.Build.TargetPath = tmpDir
	c.Build.CopySource = c.Package.CopySource
	if err = buildApp(c); err != nil {
		return
	}

	// Create the zip file.

	archiveName, err := utils.TarGzDir(destFile, tmpDir)
	if err != nil {
		return
	}

	fmt.Println("Your archive is ready:", archiveName)
	return
}
