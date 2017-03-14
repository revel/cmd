package main
import (
    "github.com/revel/cmd/revel/builder"
)

var cmdNew = &Command{
	UsageLine: "new [path] [skeleton]",
	Short:     "create a skeleton Revel application",
	Long: `
New creates a few files to get a new Revel application running quickly.

It puts all of the files in the given import path, taking the final element in
the path to be the app name.

Skeleton is an optional argument, provided as an import path

For example:

    revel new import/path/helloworld

    revel new import/path/helloworld import/path/skeleton
`,
}

func init() {
	cmdNew.Run = builder.NewSkeleton
}
