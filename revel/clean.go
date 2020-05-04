// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"

	"os"
	"path/filepath"
)

var cmdClean = &Command{
	UsageLine: "clean [import path]",
	Short:     "clean a Revel application's temp files",
	Long: `
Clean the Revel web application named by the given import path.

For example:

    revel clean github.com/revel/examples/chat

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
	if len(args) == 0 && c.Clean.ImportPath != "" {
		return true
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, cmdClean.Long)
		return false
	}
	c.Clean.ImportPath = args[0]
	return true
}

// Clean the source directory of generated files
func cleanApp(c *model.CommandConfig) (err error) {

	purgeDirs := []string{
		filepath.Join(c.AppPath, "app", "tmp"),
		filepath.Join(c.AppPath, "app", "routes"),
	}

	for _, dir := range purgeDirs {
		fmt.Println("Removing:", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			utils.Logger.Error("Failed to clean dir", "error", err)
			return
		}
	}
	return err
}
