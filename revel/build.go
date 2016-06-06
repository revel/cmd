package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/harness"
	"github.com/revel/revel"
)

var cmdBuild = &Command{
	UsageLine: "build [import path] [target path] [run mode]",
	Short:     "build a Revel application (e.g. for deployment)",
	Long: `
Build the Revel web application named by the given import path.
This allows it to be deployed and run on a machine that lacks a Go installation.

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

WARNING: The target path will be completely deleted, if it already exists!

For example:

    revel build github.com/revel/samples/chat /tmp/chat
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

	appImportPath, destPath, mode := args[0], args[1], "dev"
	if len(args) >= 3 {
		mode = args[2]
	}

	if !revel.Initialized {
		revel.Init(mode, appImportPath, "")
	}

	// First, verify that it is either already empty or looks like a previous
	// build (to avoid clobbering anything)
	if exists(destPath) && !empty(destPath) && !exists(path.Join(destPath, "run.sh")) {
		errorf("Abort: %s exists and does not look like a build directory.", destPath)
	}

	os.RemoveAll(destPath)
	os.MkdirAll(destPath, 0777)

	app, reverr := harness.Build()
	panicOnError(reverr, "Failed to build")

	// Included are:
	// - run scripts
	// - binary
	// - revel
	// - app

	// Revel and the app are in a directory structure mirroring import path
	srcPath := path.Join(destPath, "src")
	destBinaryPath := path.Join(destPath, filepath.Base(app.BinaryPath))
	tmpRevelPath := path.Join(srcPath, filepath.FromSlash(revel.REVEL_IMPORT_PATH))
	mustCopyFile(destBinaryPath, app.BinaryPath)
	mustChmod(destBinaryPath, 0755)
	mustCopyDir(path.Join(tmpRevelPath, "conf"), path.Join(revel.RevelPath, "conf"), nil)
	if revel.GO_TEMPLATE == revel.Config.StringDefault("template.engine", revel.GO_TEMPLATE) {
		mustCopyDir(path.Join(tmpRevelPath, "templates"), path.Join(revel.RevelPath, "templates"), nil)
	}
	mustCopyDir(path.Join(srcPath, filepath.FromSlash(appImportPath)), revel.BasePath, nil)

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
		mustCopyDir(path.Join(srcPath, importPath), fsPath, nil)
	}

	tmplData, runShPath := map[string]interface{}{
		"BinName":    filepath.Base(app.BinaryPath),
		"ImportPath": appImportPath,
		"Mode":       mode,
	}, path.Join(destPath, "run.sh")

	mustRenderTemplate(
		runShPath,
		filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_run.sh.template"),
		tmplData)

	mustChmod(runShPath, 0755)

	mustRenderTemplate(
		filepath.Join(destPath, "run.bat"),
		filepath.Join(revel.RevelPath, "..", "cmd", "revel", "package_run.bat.template"),
		tmplData)
}
