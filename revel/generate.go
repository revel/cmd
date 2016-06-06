package main

import (
	"fmt"
	"os"

	"github.com/revel/cmd/harness"
	"github.com/revel/revel"
)

var cmdGenerate = &Command{
	UsageLine: "generate [import path] [command path]",
	Short:     "generate a Revel application's routes and main func",
	Long: `
Generates the source code a Revel application needs to run.
This includes routing code (app/routes/routes.go) and the
main func to run the app.

Target path is a relative path from the root of your Revel app.

The generated command is expected to be run with the BasePath
and the Revel src path.

For example:

    revel generate github.com/revel/samples/chat cmd/chat
`,
}

func init() {
	cmdGenerate.Run = generateFiles
}

func generateFiles(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdBuild.UsageLine, cmdBuild.Long)
		return
	}

	appImportPath, destPath := args[0], args[1]
	if !revel.Initialized {
		revel.Init("", appImportPath, "")
	}

	reverr := harness.Generate(destPath, true)
	panicOnError(reverr, "Failed to build")
}
