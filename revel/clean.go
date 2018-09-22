// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"go/build"
	"os"
	"path/filepath"
)

var cmdClean = &Command{
	UsageLine: "clean -i [import path]",
	Short:     "clean a Revel application's temp files",
	Long: `
Clean the Revel web application named by the given import path.

For example:

    revel clean -a github.com/revel/examples/chat

It removes the app/tmp and app/routes directory.


`,
}

func init() {
	cmdClean.UpdateConfig = updateCleanConfig
	cmdClean.RunWith = cleanApp
}

// Update the clean command configuration, using old method
func updateCleanConfig(c *model.CommandConfig, args []string) bool {
	c.Index = model.CLEAN
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, cmdClean.Long)
		return false
	}
	c.Clean.ImportPath = args[0]
	return true
}

// Clean the source directory of generated files
func cleanApp(c *model.CommandConfig) {
	appPkg, err := build.Import(c.ImportPath, "", build.FindOnly)
	if err != nil {
		utils.Logger.Fatal("Abort: Failed to find import path:", "error", err)
	}

	purgeDirs := []string{
		filepath.Join(appPkg.Dir, "app", "tmp"),
		filepath.Join(appPkg.Dir, "app", "routes"),
	}

	for _, dir := range purgeDirs {
		fmt.Println("Removing:", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			utils.Logger.Error("Failed to clean dir", "error", err)
			return
		}
	}
}
