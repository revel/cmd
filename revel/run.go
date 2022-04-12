// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/revel/cmd/harness"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

var cmdRun = &Command{
	UsageLine: "run [-m [run mode] -p [port]] [import path] ",
	Short:     "run a Revel application",
	Long: `
Run the Revel web application named by the given import path.

For example, to run the chat room sample application:

    revel run -m dev github.com/revel/examples/chat

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can set a port as well.  For example:

    revel run -m prod -p 8080 github.com/revel/examples/chat `,
}

func init() {
	cmdRun.RunWith = runApp
	cmdRun.UpdateConfig = updateRunConfig
}

func updateRunConfig(c *model.CommandConfig, args []string) bool {
	convertPort := func(value string) int {
		if value != "" {
			port, err := strconv.Atoi(value)
			if err != nil {
				utils.Logger.Fatalf("Failed to parse port as integer: %s", c.Run.Port)
			}
			return port
		}
		return 0
	}
	switch len(args) {
	case 3:
		// Possible combinations
		// revel run [import-path] [run-mode] [port]
		c.Run.ImportPath = args[0]
		c.Run.Mode = args[1]
		c.Run.Port = convertPort(args[2])
	case 2:
		// Possible combinations
		// 1. revel run [import-path] [run-mode]
		// 2. revel run [import-path] [port]
		// 3. revel run [run-mode] [port]

		// Check to see if the import path evaluates out to something that may be on a gopath
		if runIsImportPath(args[0]) {
			// 1st arg is the import path
			c.Run.ImportPath = args[0]

			if _, err := strconv.Atoi(args[1]); err == nil {
				// 2nd arg is the port number
				c.Run.Port = convertPort(args[1])
			} else {
				// 2nd arg is the run mode
				c.Run.Mode = args[1]
			}
		} else {
			// 1st arg is the run mode
			c.Run.Mode = args[0]
			c.Run.Port = convertPort(args[1])
		}
	case 1:
		// Possible combinations
		// 1. revel run [import-path]
		// 2. revel run [port]
		// 3. revel run [run-mode]
		if runIsImportPath(args[0]) {
			// 1st arg is the import path
			c.Run.ImportPath = args[0]
		} else if _, err := strconv.Atoi(args[0]); err == nil {
			// 1st arg is the port number
			c.Run.Port = convertPort(args[0])
		} else {
			// 1st arg is the run mode
			c.Run.Mode = args[0]
		}
	case 0:
		// Attempt to set the import path to the current working director.
		if c.Run.ImportPath == "" {
			c.Run.ImportPath, _ = os.Getwd()
		}
	}
	c.Index = model.RUN
	return true
}

// Returns true if this is an absolute path or a relative gopath.
func runIsImportPath(pathToCheck string) bool {
	return utils.DirExists(pathToCheck)
}

// Called to run the app.
func runApp(c *model.CommandConfig) (err error) {
	if c.Run.Mode == "" {
		c.Run.Mode = "dev"
	}

	revelPath, err := model.NewRevelPaths(c.Run.Mode, c.ImportPath, c.AppPath, model.NewWrappedRevelCallback(nil, c.PackageResolver))
	if err != nil {
		return utils.NewBuildIfError(err, "Revel paths")
	}
	if c.Run.Port > -1 {
		revelPath.HTTPPort = c.Run.Port
	} else {
		c.Run.Port = revelPath.HTTPPort
	}

	utils.Logger.Infof("Running %s (%s) in %s mode\n", revelPath.AppName, revelPath.ImportPath, revelPath.RunMode)
	utils.Logger.Debug("Base path:", "path", revelPath.BasePath)

	// If the app is run in "watched" mode, use the harness to run it.
	if revelPath.Config.BoolDefault("watch", true) && revelPath.Config.BoolDefault("watch.code", true) {
		utils.Logger.Info("Running in watched mode.")

		runMode := fmt.Sprintf(`{"mode":"%s", "specialUseFlag":%v}`, revelPath.RunMode, c.GetVerbose())
		if c.HistoricMode {
			runMode = revelPath.RunMode
		}
		// **** Never returns.
		harness.NewHarness(c, revelPath, runMode, c.Run.NoProxy).Run()
	}

	// Else, just build and run the app.
	utils.Logger.Debug("Running in live build mode.")
	app, err := harness.Build(c, revelPath)
	if err != nil {
		utils.Logger.Errorf("Failed to build app: %s", err)
	}
	app.Port = revelPath.HTTPPort
	var paths []byte
	if len(app.PackagePathMap) > 0 {
		paths, _ = json.Marshal(app.PackagePathMap)
	}
	runMode := fmt.Sprintf(`{"mode":"%s", "specialUseFlag":%v,"packagePathMap":%s}`, app.Paths.RunMode, c.GetVerbose(), string(paths))
	if c.HistoricMode {
		runMode = revelPath.RunMode
	}
	app.Cmd(runMode).Run(c)
	return
}
