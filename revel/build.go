// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/revel/cmd/harness"
    "github.com/revel/cmd/revel/util"
    "github.com/revel/revel"
)

var cmdBuild = &Command{
    UsageLine: "build [import path] [target path] [run mode] [Include source]",
    Short:     "build a Revel application (e.g. for deployment)",
    Long: `
Build the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

WARNING: The target path will be completely deleted, if it already exists!

For example:

    revel build github.com/revel/examples/chat /tmp/chat
`,
}

func init() {
    cmdBuild.Run = buildApp
}

func buildApp(args []string) {
    if len(args) < 2 {
        fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
        return
    }

    appImportPath, destPath, mode := args[0], args[1], util.DefaultRunMode
    if len(args) >= 3 {
        mode = args[2]
    }
    buildTheApp(appImportPath, destPath, mode)
}

func buildTheApp(appImportPath, destPath, mode string) {

    if !revel.Initialized {
        revel.Init(mode, appImportPath, "")
    }

    // First, verify that it is either already empty or looks like a previous
    // build (to avoid clobbering anything)
    if util.Exists(destPath) && !util.Empty(destPath) && !util.Exists(filepath.Join(destPath, "run.sh")) {
        util.Errorf("Abort: %s exists and does not look like a build directory.", destPath)
    }

    if err := os.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
        revel.ERROR.Fatalln(err)
    }

    if err := os.MkdirAll(destPath, 0777); err != nil {
        revel.ERROR.Fatalln(err)
    }

    app, reverr := harness.Build()
    util.PanicOnError(reverr, "Failed to build")

    // Included are:
    // - run scripts
    // - binary
    // - revel
    // - app

    // Revel and the app are in a directory structure mirroring import path
    srcPath := filepath.Join(destPath, "src")
    destBinaryPath := filepath.Join(destPath, filepath.Base(app.BinaryPath))
    tmpRevelPath := filepath.Join(srcPath, filepath.FromSlash(revel.RevelImportPath))
    util.MustCopyFile(destBinaryPath, app.BinaryPath)
    util.MustChmod(destBinaryPath, 0755)
    _ = util.MustCopyDir(filepath.Join(tmpRevelPath, "conf"), filepath.Join(revel.RevelPath, "conf"), nil)
    _ = util.MustCopyDir(filepath.Join(tmpRevelPath, "templates"), filepath.Join(revel.RevelPath, "templates"), nil)
    _ = util.MustCopyDir(filepath.Join(srcPath, filepath.FromSlash(appImportPath)), revel.BasePath, nil)

    // Find all the modules used and copy them over.
    config := revel.Config.Raw()
    modulePaths := make(map[string]string) // import path => filesystem path
    for _, section := range config.Sections() {
        options, _ := config.SectionOptions(section)
        for _, key := range options {
            if !strings.HasPrefix(key, "module.") {
                continue
            }
            moduleImportPath, _ := config.String(section, key)
            if moduleImportPath == "" {
                continue
            }
            modulePath, err := revel.ResolveImportPath(moduleImportPath)
            if err != nil {
                revel.ERROR.Fatalln("Failed to load module %s: %s", key[len("module."):], err)
            }
            modulePaths[moduleImportPath] = modulePath
        }
    }
    for importPath, fsPath := range modulePaths {
        _ = util.MustCopyDir(filepath.Join(srcPath, importPath), fsPath, nil)
    }

    tmplData, runShPath := map[string]interface{}{
        "BinName":    filepath.Base(app.BinaryPath),
        "ImportPath": appImportPath,
        "Mode":       mode,
    }, filepath.Join(destPath, "run.sh")

    util.MustRenderTemplate(
        runShPath,
        filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_run.sh.template"),
        tmplData)

    util.MustChmod(runShPath, 0755)

    util.MustRenderTemplate(
        filepath.Join(destPath, "run.bat"),
        filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_run.bat.template"),
        tmplData)
}
