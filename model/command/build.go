package command

type (
	Build struct {
		ImportCommand
		TargetPath string `short:"t" long:"target-path" description:"Path to target folder. Folder will be completely deleted if it exists" required:"false"`
		Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
		CopySource bool   `short:"s" long:"include-source" description:"Copy the source code as well"`
	}
)
