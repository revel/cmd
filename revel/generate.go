// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/revel/cmd/harness"
	"github.com/revel/revel"
)

var cmdGenerate = &Command{
	UsageLine: "generate [import path] [source path] [find] [replace]",
	Short:     "Generate the routes and main.go files",
	Long: `
Generate the routes and main.go files for a revel project.

This is necessary to build applications that are not organized in the default
GOPATH. Because we may not be in the default GOPATH, some of the imports in
main.go may have to be rewritten. Use the optional find and replace to do this.

WARNING: the app/routes and app/tmp directories will be deleted.

For example:

    revel generate github.com/revel/examples/chat ./

    revel generate revel/examples/chat ./ revel/examples github.com/revel/examples
`,
}

func init() {
	cmdGenerate.Run = generateRoutesMain
}

func generateRoutesMain(args []string) {
	if len(args) < 2 {
		errMsg := "Import and source paths are required."
		fmt.Fprintf(os.Stderr, "%s\n%s\n%s", errMsg, cmdGenerate.UsageLine, cmdGenerate.Long)
		return
	}

	appImportPath, srcPath := args[0], args[1]

	var find, replace string
	if len(args) >= 3 {
		if len(args) < 4 {
			errMsg := "Specify a replacement string if find string specified."
			fmt.Fprintf(os.Stderr, "%s\n%s\n%s", errMsg, cmdBuild.UsageLine, cmdBuild.Long)
			return
		}
		find, replace = args[2], args[3]
	}

	if !revel.Initialized {
		revel.Init("", appImportPath, srcPath)
	}

	reverr := harness.GenerateRoutesMain(revel.CodePaths, true, find, replace)
	panicOnError(reverr, "Failed to generate files")
}
