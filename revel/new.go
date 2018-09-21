// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/build"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

var cmdNew = &Command{
	UsageLine: "new -i [path] -s [skeleton]",
	Short:     "create a skeleton Revel application",
	Long: `
New creates a few files to get a new Revel application running quickly.

It puts all of the files in the given import path, taking the final element in
the path to be the app name.

Skeleton is an optional argument, provided as an import path

For example:

    revel new -a import/path/helloworld

    revel new -a import/path/helloworld -s import/path/skeleton

`,
}

func init() {
	cmdNew.RunWith = newApp
	cmdNew.UpdateConfig = updateNewConfig
}

// Called when unable to parse the command line automatically and assumes an old launch
func updateNewConfig(c *model.CommandConfig, args []string) bool {
	c.Index = NEW
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, cmdNew.Long)
		return false
	}
	c.New.ImportPath = args[0]
	if len(args)>1 {
		c.New.Skeleton = args[1]
	}
	return true

}

// Call to create a new application
func newApp(c *model.CommandConfig) {
	// check for proper args by count
	c.ImportPath = c.New.ImportPath
	c.SkeletonPath = c.New.Skeleton

	// Check for an existing folder so we dont clober it
	c.AppPath = filepath.Join(c.SrcRoot, filepath.FromSlash(c.ImportPath))
	_, err := build.Import(c.ImportPath, "", build.FindOnly)
	if err==nil || !utils.Empty(c.AppPath) {
		utils.Logger.Fatal("Abort: Import path already exists.","path", c.ImportPath)
	}

	if c.New.Vendored {
		depPath, err := exec.LookPath("dep")
		if err != nil {
			// Do not halt build unless a new package needs to be imported
			utils.Logger.Fatal("New: `dep` executable not found in PATH, but vendor folder requested." +
				"You must install the dep tool before creating a vendored project. " +
				"You can install the `dep` tool by doing a `go get -u github.com/golang/dep/cmd/dep`")
		}
		vendorPath := filepath.Join(c.ImportPath,"vendor")
		if !utils.DirExists(vendorPath) {
			err := os.MkdirAll(vendorPath,os.ModePerm)
			utils.PanicOnError(err, "Failed to create " + vendorPath)
		}
		// In order for dep to run there needs to be a source file in the folder
		tempPath := filepath.Join(c.ImportPath,"tmp")
		if !utils.DirExists(tempPath) {
			err := os.MkdirAll(tempPath,os.ModePerm)
			utils.PanicOnError(err, "Failed to create " + vendorPath)
			err = utils.MustGenerateTemplate(filepath.Join(tempPath,"main.go"), NEW_MAIN_FILE, nil)
			utils.PanicOnError(err, "Failed to create main file " + vendorPath)

		}
		packageFile := filepath.Join(c.ImportPath,"Gopkg.toml")
		if !utils.Exists(packageFile) {
			utils.MustGenerateTemplate(packageFile,VENDOR_GOPKG,nil)
		} else {
			utils.Logger.Info("Package file exists in skeleto, skipping adding")
		}

		getCmd := exec.Command(depPath, "ensure", "-v")
		getCmd.Dir = c.ImportPath
		utils.Logger.Info("Exec:", "args", getCmd.Args)
		getCmd.Dir = c.ImportPath
		getOutput, err := getCmd.CombinedOutput()
		if err != nil {
			utils.Logger.Fatal(string(getOutput))
		}
	}


	// checking and setting application
	setApplicationPath(c)

	// checking and setting skeleton
	setSkeletonPath(c)

	// copy files to new app directory
	copyNewAppFiles(c)


	// goodbye world
	fmt.Fprintln(os.Stdout, "Your application is ready:\n  ", c.AppPath)
	// Check to see if it should be run right off
	if c.New.Run {
		c.Run.ImportPath = c.ImportPath
		runApp(c)
	} else {
		fmt.Fprintln(os.Stdout, "\nYou can run it with:\n   revel run -a ", c.ImportPath)
	}
}

// Used to generate a new secret key
const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// Generate a secret key
func generateSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}

// Sets the applicaiton path
func setApplicationPath(c *model.CommandConfig) {

	// revel/revel#1014 validate relative path, we cannot use built-in functions
	// since Go import path is valid relative path too.
	// so check basic part of the path, which is "."
	if filepath.IsAbs(c.ImportPath) || strings.HasPrefix(c.ImportPath, ".") {
		utils.Logger.Fatalf("Abort: '%s' looks like a directory.  Please provide a Go import path instead.",
			c.ImportPath)
	}


	// If we are running a vendored version of Revel we do not need to check for it.
	if !c.New.Vendored {
		var err error
		_, err = build.Import(model.RevelImportPath, "", build.FindOnly)
		if err != nil {
			// Go get the revel project
			getCmd := exec.Command(c.GoCmd, "get", model.RevelImportPath)
			utils.Logger.Info("Exec:" + c.GoCmd, "args", getCmd.Args)
			getOutput, err := getCmd.CombinedOutput()
			if err != nil {
				utils.Logger.Fatal("Failed to fetch revel " + model.RevelImportPath, "getOutput", string(getOutput))
			}
		}
	}

	c.AppName = filepath.Base(c.AppPath)
	c.BasePath = filepath.ToSlash(filepath.Dir(c.ImportPath))

	if c.BasePath == "." {
		// we need to remove the a single '.' when
		// the app is in the $GOROOT/src directory
		c.BasePath = ""
	} else {
		// we need to append a '/' when the app is
		// is a subdirectory such as $GOROOT/src/path/to/revelapp
		c.BasePath += "/"
	}
}

// Set the skeleton path
func setSkeletonPath(c *model.CommandConfig) {
	var err error
	if len(c.SkeletonPath) > 0 { // user specified

		_, err = build.Import(c.SkeletonPath, "", build.FindOnly)
		if err != nil {
			// Execute "go get <pkg>"
			getCmd := exec.Command(c.GoCmd, "get", "-d", c.SkeletonPath)
			fmt.Println("Exec:", getCmd.Args)
			getOutput, err := getCmd.CombinedOutput()

			// check getOutput for no buildible string
			bpos := bytes.Index(getOutput, []byte("no buildable Go source files in"))
			if err != nil && bpos == -1 {
				utils.Logger.Fatalf("Abort: Could not find or 'go get' Skeleton  source code: %s\n%s\n", getOutput, c.SkeletonPath)
			}
		}
		// use the
		c.SkeletonPath = filepath.Join(c.SrcRoot, c.SkeletonPath)

	} else {
		// use the revel default
		revelCmdPkg, err := build.Import(RevelCmdImportPath, "", build.FindOnly)
		if err != nil {
			if err != nil {
				// Go get the revel project
				getCmd := exec.Command(c.GoCmd, "get", RevelCmdImportPath + "/revel")
				utils.Logger.Info("Exec:" + c.GoCmd, "args", getCmd.Args)
				getOutput, err := getCmd.CombinedOutput()
				if err != nil {
					utils.Logger.Fatal("Failed to fetch revel cmd " + RevelCmdImportPath, "getOutput", string(getOutput))
				}
				revelCmdPkg, err = build.Import(RevelCmdImportPath, "", build.FindOnly)
				if err!= nil {
					utils.Logger.Fatal("Failed to find source of revel cmd " + RevelCmdImportPath, "getOutput", string(getOutput), "error",err, "dir", revelCmdPkg.Dir)
				}
			}
		}

		c.SkeletonPath = filepath.Join(revelCmdPkg.Dir, "revel", "skeleton")
	}
}

func copyNewAppFiles(c *model.CommandConfig) {
	var err error
	err = os.MkdirAll(c.AppPath, 0777)
	utils.PanicOnError(err, "Failed to create directory "+c.AppPath)

	_ = utils.MustCopyDir(c.AppPath, c.SkeletonPath, map[string]interface{}{
		// app.conf
		"AppName":  c.AppName,
		"BasePath": c.BasePath,
		"Secret":   generateSecret(),
	})

	// Dotfiles are skipped by mustCopyDir, so we have to explicitly copy the .gitignore.
	gitignore := ".gitignore"
	utils.MustCopyFile(filepath.Join(c.AppPath, gitignore), filepath.Join(c.SkeletonPath, gitignore))

}

const (
	VENDOR_GOPKG = `#
# Revel Gopkg.toml
#
# If you want to use a specific version of Revel change the branches below
#
# Refer to https://github.com/golang/dep/blob/master/docs/Gopkg.toml.md
# for detailed Gopkg.toml documentation.
#
# required = ["github.com/user/thing/cmd/thing"]
# ignored = ["github.com/user/project/pkgX", "bitbucket.org/user/project/pkgA/pkgY"]
#
# [[constraint]]
#   name = "github.com/user/project"
#   version = "1.0.0"
#
# [[constraint]]
#   name = "github.com/user/project2"
#   branch = "dev"
#   source = "github.com/myfork/project2"
#
# [[override]]
#  name = "github.com/x/y"
#  version = "2.4.0"
required = ["github.com/revel/cmd/revel"]

[[override]]
  branch = "master"
  name = "github.com/revel/modules"

[[override]]
  branch = "master"
  name = "github.com/revel/revel"

[[override]]
  branch = "master"
  name = "github.com/revel/cmd"

[[override]]
  branch = "master"
  name = "github.com/revel/log15"

[[override]]
  branch = "master"
  name = "github.com/revel/cron"

[[override]]
  branch = "master"
  name = "github.com/xeonx/timeago"

`
	NEW_MAIN_FILE = `package main

	`
)