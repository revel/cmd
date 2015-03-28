package main

import (
	"fmt"
	"go/build"
	"os"
	"path"
)

var cmdClean = &Command{
	UsageLine: "clean [import path]",
	Short:     "clean a Revel application's temp files",
	Long: `
Clean the Revel web application named by the given import path.

For example:

    revel clean github.com/revel/samples/chat

It removes the app/tmp directory.
`,
}

func init() {
	cmdClean.Run = cleanApp
}

func cleanApp(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, cmdClean.Long)
		return
	}

	appPkg, err := build.Import(args[0], "", build.FindOnly)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Abort: Failed to find import path:", err)
		return
	}

	// Remove the app/tmp directory.
	tmpDir := path.Join(appPkg.Dir, "app", "tmp")
	clearPath(tmpDir)

	// Remove the public/assets directory.
	assetsPath := path.Join(appPkg.Dir, "public", "assets")
	clearPath(assetsPath)
}

func clearPath(path string) {
	fmt.Println("Removing:", path)
	err := os.RemoveAll(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Abort:", err)
		return
	}
}
