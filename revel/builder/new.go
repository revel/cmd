// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package builder

import (
	"fmt"
	"go/build"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/revel/revel"
	"io/ioutil"
)



var (


	// revel related paths
	revelPkg     *build.Package
	revelCmdPkg  *build.Package
)
type NewConfig struct {
	// go related paths
	gopath  string
	gocmd   string
	srcRoot string
	appPath      string
	appName      string
	basePath     string
	importPath   string
	skeletonPath string
}
type BaseNewApp struct {
    Config *NewConfig
}
func(i *BaseNewApp) newApp(args []string) {
	// check for proper args by count
	if len(args) == 0 {
		i.errorf("No import path given.\nRun 'revel help new' for usage.\n")
	}
	if len(args) > 2 {
		i.errorf("Too many arguments provided.\nRun 'revel help new' for usage.\n")
	}

	revel.ERROR.SetFlags(log.LstdFlags)

	// checking and setting go paths

    i.Config.importPath = args[0]
	i.initGoPaths()

	// checking and setting application
	i.setApplicationPath()

}

const alphaNumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func(i *BaseNewApp)  generateSecret() string {
	chars := make([]byte, 64)
	for i := 0; i < 64; i++ {
		chars[i] = alphaNumeric[rand.Intn(len(alphaNumeric))]
	}
	return string(chars)
}

// lookup and set Go related variables
func(i *BaseNewApp)  initGoPaths() {
	// lookup go path
	i.Config.gopath = build.Default.GOPATH
	if i.Config.gopath == "" {
		i.errorf("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}
	if i.Config.importPath == "" {
		i.errorf("Empty application path, please specify an application. ")
	}

	// check for go executable
	var err error
	i.Config.gocmd, err = exec.LookPath("go")
	if err != nil {
		i.errorf("Go executable not found in PATH.")
	}

	// revel/revel#1004 choose go path relative to current working directory
	workingDir, _ := os.Getwd()
	workingDir, _ = filepath.EvalSymlinks(workingDir)
	targetDir := filepath.Join(workingDir, i.Config.importPath)
	// Check to see if files exists in working folder
	if _, err := os.Stat(targetDir); err == nil {
		revel.ERROR.Fatalf("Revel application already exists in gopath %s.", filepath.Join(workingDir, i.Config.importPath))
	}
	goPathList := filepath.SplitList(i.Config.gopath)

	foundGopath := false
	for _, path := range goPathList {
		path, _ = filepath.EvalSymlinks(path)

		if len(path) > 0 && strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			// Now the full path may be missing some parts to it since the working dir might be different
			i.Config.srcRoot = path
			foundGopath = true
			break
		}
	}
	if !foundGopath {
		// Extended search, create the directory
		err = os.MkdirAll(targetDir, os.ModePerm)
		if err != nil {
			revel.ERROR.Fatalf("Unable to create target dir %s %s.", targetDir, err)
		}
		defer func() {
			for {
				os.Remove(targetDir)
				targetDir = filepath.Dir(targetDir)
				testDir := filepath.Dir(targetDir)
				if !strings.HasPrefix(testDir, workingDir) {
					break
				}
			}
		}()
		tempFile, err := ioutil.TempFile(targetDir, "test")
		if err != nil {
			revel.ERROR.Fatalf("Unable to create target file %s %s.", targetDir, err)
		}
		defer func() {
			tempFile.Close()
			os.Remove(tempFile.Name())
		}()
		checkfile := filepath.Base(tempFile.Name())
		for _, path := range goPathList {
			// Walk down the filepath to see if the temp file can be found
			cont, foundpath, _ := i.fullFileWalk(filepath.Join(path, "src"), checkfile, 0)
			if !cont {
                // There may be a directory or two between the source and the folder placement.
				// In theory this should still be ok
                i.Config.srcRoot = path
                i.Config.appPath = filepath.Dir(foundpath)
                // Import path needs to be updated Now
                i.Config.importPath = i.Config.appPath[len(path)+1:]
                foundGopath = true

                break
			}
		}
	}

	if !foundGopath {
		i.errorf("Revel application may be outside of GOPATH.")
	}

	// set go src path
	i.Config.srcRoot = filepath.Join(i.Config.srcRoot, "src")
}

func(i *BaseNewApp)  fullFileWalk(path, checkfile string, depth int) (cont bool, foundpath string, err error) {
	path, _ = filepath.EvalSymlinks(path)
	dir, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}
	for _, file := range dir {
		fullName := filepath.Join(path, file.Name())
		// It is only the symlink that will mess up the path
		if !file.IsDir() {
			realPath, e := filepath.EvalSymlinks(fullName)
			if e != nil {
				// Broken symlink, ignore
				continue
			}
			if realPath != fullName {
				file, _ = os.Stat(realPath)
			}
		}
		if file.IsDir() && depth < 6 {
			cont, foundpath, err = i.fullFileWalk(fullName, checkfile, depth+1)
			if !cont || err != nil {
				return
			}
			continue
		}
		if file.Name() == checkfile {
			foundpath = filepath.Join(path, file.Name())
			return false, foundpath, nil
		}
	}
	cont = true
	return
}

func(i *BaseNewApp)  setApplicationPath() {
	var err error

	// revel/revel#1014 validate relative path, we cannot use built-in functions
	// since Go import path is valid relative path too.
	// so check basic part of the path, which is "."
	if filepath.IsAbs(i.Config.importPath) || strings.HasPrefix(i.Config.importPath, ".") {
		i.errorf("Abort: '%s' looks like a directory.  Please provide a Go import path instead.",
			i.Config.importPath)
	}

	_, err = build.Import(i.Config.importPath, "", build.FindOnly)
	if err == nil {
		i.errorf("Abort: Import path %s already exists.\n", i.Config.importPath)
	}

	revelPkg, err = build.Import(revel.RevelImportPath, "", build.FindOnly)
	if err != nil {
		i.errorf("Abort: Could not find Revel source code: %s\n", err)
	}

	i.Config.appPath = filepath.Join(i.Config.srcRoot, filepath.FromSlash(i.Config.importPath))
	i.Config.appName = filepath.Base(i.Config.appPath)
	i.Config.basePath = filepath.ToSlash(filepath.Dir(i.Config.importPath))

	if i.Config.basePath == "." {
		// we need to remove the a single '.' when
		// the app is in the $GOROOT/src directory
		i.Config.basePath = ""
	} else {
		// we need to append a '/' when the app is
		// is a subdirectory such as $GOROOT/src/path/to/revelapp
		i.Config.basePath += "/"
	}
}



func (i *BaseNewApp) errorf(format string, args ...interface{}) {
	// Ensure the user's command prompt starts on the next line.
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	panic("Halted") // Panic instead of os.Exit so that deferred will run.
}
