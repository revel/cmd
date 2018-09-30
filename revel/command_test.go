package main_test

import (
	"github.com/revel/cmd/logger"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"github.com/stretchr/testify/assert"
	"go/build"
	"os"
	"path/filepath"
)

// Test that the event handler can be attached and it dispatches the event received
func setup(suffix string, a *assert.Assertions) (string) {
	temp := os.TempDir()
	wd, _ := os.Getwd()
	utils.InitLogger(wd, logger.LvlInfo)
	gopath := filepath.Join(temp, "revel-test",suffix)
	if utils.Exists(gopath) {
		utils.Logger.Info("Removing test path", "path", gopath)
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
	err := os.MkdirAll(gopath, os.ModePerm)
	a.Nil(err, "Failed to create gopath "+gopath)

	// So this is the issue, on the mac when folders are created in a temp folder they are returned like
	// /var/folders/nz/vv4_9tw56nv9k3tkvyszvwg80000gn/T/revel-test/revel-test-build
	// But if you change into that directory and read the current folder it is
	// /private/var/folders/nz/vv4_9tw56nv9k3tkvyszvwg80000gn/T/revel-test/revel-test-build
	// So to make this work on darwin this code was added
	os.Chdir(gopath)
	newwd, _ := os.Getwd()
	gopath = newwd
	defaultBuild := build.Default
	defaultBuild.GOPATH = gopath
	build.Default = defaultBuild
	utils.Logger.Info("Setup stats", "original wd", wd, "new wd", newwd, "gopath",gopath, "gopath exists", utils.DirExists(gopath), "wd exists", utils.DirExists(newwd))

	return gopath
}

// Create a new app for the name
func newApp(name string, command model.COMMAND, precall func(c *model.CommandConfig), a *assert.Assertions) *model.CommandConfig {
	c := &model.CommandConfig{}
	switch command {
	case model.NEW:
		c.New.ImportPath = name
	case model.BUILD:
		c.Build.ImportPath = name
	case model.TEST:
		c.Test.ImportPath = name
	case model.PACKAGE:
		c.Package.ImportPath = name
	case model.VERSION:
		c.Version.ImportPath = name
	case model.CLEAN:
		c.Clean.ImportPath = name
	default:
		a.Fail("Unknown command ", command)
	}

	c.Index = command
	if precall != nil {
		precall(c)
	}
	if !c.UpdateImportPath() {
		a.Fail("Unable to update import path")
	}
	c.InitGoPaths()
	c.InitPackageResolver()
	return c
}
