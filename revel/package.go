// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/revel/revel"
    "github.com/revel/cmd/revel/util"
    "strconv"
    "strings"
)

var cmdPackage = &Command{
    UsageLine: "package [import path] [run mode] [include source]",
    Short:     "package a Revel application (e.g. for deployment)",
    Long: `
Package the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

The include source arguement defaults to true, if you enter false
only paths under views/** messages/** conf/** public/** will be included

Note symlinked forlders are not included

Run mode defaults to "dev".

For example:

    revel package github.com/revel/examples/chat prod false
`,
}

func init() {
    cmdPackage.Run = packageApp
}

func packageApp(args []string) {
    if len(args) == 0 {
        fmt.Fprint(os.Stderr, cmdPackage.Long)
        return
    }

    // Determine the run mode.
    mode := util.DefaultRunMode
    if len(args) >= 2 {
        mode = args[1]
    }
    appImportPath := args[0]
    revel.Init(mode, appImportPath, "")

    // Remove the archive if it already exists.
    destFile := filepath.Base(revel.BasePath) + ".tar.gz"
    if err := os.Remove(destFile); err != nil && !os.IsNotExist(err) {
        revel.ERROR.Fatal(err)
    }

    // Collect stuff in a temp directory.
    // tmpDir, err := ioutil.TempDir("", filepath.Base(revel.BasePath))
    //tmpDir,err := filepath.Join(os.Getwd(),"rbuild"), os.Mkdir("rbuild",os.ModePerm)
    //println("Created ",tmpDir)
    //util.PanicOnError(err, "Failed to get temp dir")
    wd ,_ := os.Getwd()
    tmpDir := filepath.Join(wd,"rbuild")

    includeSource := func(file string) bool{ return true }
    if len(args) >= 3 {
        if ok,_ := strconv.ParseBool(args[2]);!ok {
            includeSource = func(file string) bool {
                file = filepath.ToSlash(file)
                shortPath := file[len(tmpDir)+1:]
                return strings.Contains(file,"/views/") ||
                    strings.Contains(file,"/public/") ||
                    strings.Contains(file,"/conf/") ||
                    strings.Contains(file,"/messages/") ||
                    strings.Contains(file,"/templates/") || // Revel has these
                    !strings.Contains(shortPath,"/")

            }
        }

    }



    buildTheApp(args[0],tmpDir,mode)

    // Create the zip file.
    archiveName := util.MustTarGzDir(destFile, tmpDir, includeSource)

    fmt.Println("Your archive is ready:", archiveName)
}
