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
	"github.com/revel/revel"
)

var cmdBuildMinify = &Command{
	UsageLine: "build-minify [import path] [target path] [run mode]",
	Short:     "build minify a Revel application (e.g. for deployment)",
	Long: `
Build the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

WARNING: The target path will be completely deleted, if it already exists!

For example:

    revel build-minify github.com/revel/examples/chat /tmp/chat
`,
}

func init() {
	cmdBuildMinify.Run = buildMinifyApp
}

func buildMinifyApp(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuildMinify.UsageLine, cmdBuildMinify.Long)
		return
	}

	appImportPath, destPath, mode := args[0], args[1], DefaultRunMode
	if len(args) >= 3 {
		mode = args[2]
	}

	if !revel.Initialized {
		revel.Init(mode, appImportPath, "")
	}

	// First, verify that it is either already empty or looks like a previous
	// build (to avoid clobbering anything)
	if exists(destPath) && !empty(destPath) && !exists(filepath.Join(destPath, "run.sh")) {
		errorf("Abort: %s exists and does not look like a build directory.", destPath)
	}

	if err := os.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
		revel.RevelLog.Fatal("Remove all error","error", err)
	}

	if err := os.MkdirAll(destPath, 0777); err != nil {
		revel.RevelLog.Fatal("makedir error","error",err)
	}

	app, reverr := harness.Build()
	panicOnError(reverr, "Failed to build")

	// Included are:
	// - run scripts
	// - binary
	// - revel
	// - app

	// Revel and the app are in a directory structure mirroring import path
	destBinaryPath := filepath.Join(destPath, filepath.Base(app.BinaryPath))
    tmpRevelPath := filepath.Join(destPath, filepath.FromSlash(revel.RevelImportPath))
	mustCopyFile(destBinaryPath, app.BinaryPath)
	mustChmod(destBinaryPath, 0755)
	_ = mustCopyDir(filepath.Join(tmpRevelPath, "conf"), filepath.Join(revel.RevelPath, "conf"), nil)
	_ = mustCopyDir(filepath.Join(tmpRevelPath, "templates"), filepath.Join(revel.RevelPath, "templates"), nil)
    // default skip build objs Makefile
	_ = mustCopyDir(filepath.Join(destPath), revel.BasePath, nil,"build","objs","Makefile","log","tmp","temp","docs","docker","Dockerfile","docker-compose")

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
				revel.RevelLog.Fatalf("Failed to load module %s: %s", key[len("module."):], err)
			}
			modulePaths[moduleImportPath] = modulePath
		}
	}
	for importPath, fsPath := range modulePaths {
		_ = mustCopyDir(filepath.Join(destPath, importPath), fsPath, nil)
	}

	tmplData, runShPath := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
		"Mode":       mode,
	}, filepath.Join(destPath, "run.sh")

	mustRenderTemplate(
		runShPath,
		filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_minify_run.sh.template"),
		tmplData)

	mustChmod(runShPath, 0755)

	mustRenderTemplate(
		filepath.Join(destPath, "run.bat"),
		filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_minify_run.bat.template"),
		tmplData)

	// remove all .go file and empty dir
	filepath.Walk(destPath, func(destPath string, info os.FileInfo, err error) error {
		if strings.HasSuffix(destPath, ".go") {
			os.Remove(destPath)
		}
		return nil
	})

	filepath.Walk(destPath, func(destPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			os.Remove(destPath)
		}
		return nil
	})
}
