package command
type (
	Package struct {
		ImportCommand
		TargetPath string `short:"t" long:"target-path" description:"Full path and filename of target package to deploy" required:"false"`
		Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
		CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
	}
)