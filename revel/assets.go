package main

import (
	"fmt"
	. "github.com/huacnlee/train/command"
	"github.com/revel/revel"
	"os"
)

var cmdAssets = &Command{
	UsageLine: "assets",
	Short:     "compile assets to public/assets",
	Long: `
This command will compile files in app/assets from scss -> css, coffee -> js ...
More info: github.com/huacnlee/train

For example:

    revel assets github.com/revel/samples/chat
`,
}

func init() {
	cmdAssets.Run = compileAssets
}

func compileAssets(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "%s\n%s", cmdAssets.UsageLine, cmdAssets.Long)
		return
	}

	appImportPath := args[0]
	revel.Init("", appImportPath, "")

	// Remove the archive if it already exists.
	destFile := revel.BasePath + "/public"
	os.Remove(destFile)

	assetsPath := revel.BasePath + "/app/assets"
	fmt.Println("Compiling ", assetsPath)
	Bundle(assetsPath, destFile)

	fmt.Println("Assets Compile successed.")
}
