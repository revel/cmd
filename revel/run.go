// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"strconv"

	"fmt"
	"github.com/revel/cmd/harness"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"go/build"
)

var cmdRun = &Command{
	UsageLine: "run [import path] [run mode] [port]",
	Short:     "run a Revel application",
	Long: `
Run the Revel web application named by the given import path.

For example, to run the chat room sample application:

    revel run github.com/revel/examples/chat dev

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can set a port as an optional third parameter.  For example:

    revel run github.com/revel/examples/chat prod 8080`,
}

// RunArgs holds revel run parameters
type RunArgs struct {
	ImportPath string
	Mode       string
	Port       int
}

func init() {
	cmdRun.RunWith = runApp
	cmdRun.UpdateConfig = updateRunConfig
}

func updateRunConfig(c *model.CommandConfig, args []string) bool {

	switch len(args) {
	case 3:
		// Possible combinations
		// revel run [import-path] [run-mode] [port]
		c.Run.ImportPath = args[0]
		c.Run.Mode = args[1]
		c.Run.Port = args[2]
	case 2:
		// Possible combinations
		// 1. revel run [import-path] [run-mode]
		// 2. revel run [import-path] [port]
		// 3. revel run [run-mode] [port]

		// Check to see if the import path evaluates out to something that may be on a gopath
		if _, err := build.Import(args[0], "", build.FindOnly); err == nil {
			// 1st arg is the import path
			c.Run.ImportPath = args[0]

			if _, err := strconv.Atoi(args[1]); err == nil {
				// 2nd arg is the port number
				c.Run.Port = args[1]
			} else {
				// 2nd arg is the run mode
				c.Run.Mode = args[1]
			}
		} else {
			// 1st arg is the run mode
			c.Run.Mode = args[0]
			c.Run.Port = args[1]
		}
	case 1:
		// Possible combinations
		// 1. revel run [import-path]
		// 2. revel run [port]
		// 3. revel run [run-mode]
		_, err := build.Import(args[0], "", build.FindOnly)
		if err != nil {
			utils.Logger.Warn("Unable to run using an import path, assuming import path is working directory %s %s", "Argument", args[0], "error", err.Error())
		}
		utils.Logger.Info("Trying to build with", args[0], err)
		if err == nil {
			// 1st arg is the import path
			c.Run.ImportPath = args[0]
		} else if _, err := strconv.Atoi(args[0]); err == nil {
			// 1st arg is the port number
			c.Run.Port = args[0]
		} else {
			// 1st arg is the run mode
			c.Run.Mode = args[0]
		}
	case 0:
		return false
	}
	c.Index = model.RUN
	return true
}

func runApp(c *model.CommandConfig) {
	if c.Run.Mode == "" {
		c.Run.Mode = "dev"
	}

	revel_path := model.NewRevelPaths(c.Run.Mode, c.ImportPath, "", model.DoNothingRevelCallback)
	if c.Run.Port != "" {
		port, err := strconv.Atoi(c.Run.Port)
		if err != nil {
			utils.Logger.Fatalf("Failed to parse port as integer: %s", c.Run.Port)
		}
		revel_path.HTTPPort = port
	}

	utils.Logger.Infof("Running %s (%s) in %s mode\n", revel_path.AppName, revel_path.ImportPath, revel_path.RunMode)
	utils.Logger.Debug("Base path:", "path", revel_path.BasePath)

	// If the app is run in "watched" mode, use the harness to run it.
	if revel_path.Config.BoolDefault("watch", true) && revel_path.Config.BoolDefault("watch.code", true) {
		utils.Logger.Info("Running in watched mode.")
		runMode := fmt.Sprintf(`{"mode":"%s", "specialUseFlag":%v}`, revel_path.RunMode, c.Verbose)
		if c.HistoricMode {
			runMode = revel_path.RunMode
		}
		// **** Never returns.
		harness.NewHarness(c, revel_path, runMode, c.Run.NoProxy).Run()
	}

	// Else, just build and run the app.
	utils.Logger.Debug("Running in live build mode.")
	app, err := harness.Build(c, revel_path)
	if err != nil {
		utils.Logger.Errorf("Failed to build app: %s", err)
	}
	app.Port = revel_path.HTTPPort
	runMode := fmt.Sprintf(`{"mode":"%s", "specialUseFlag":%v}`, app.Paths.RunMode, c.Verbose)
	if c.HistoricMode {
		runMode = revel_path.RunMode
	}
	app.Cmd(runMode).Run()
}
