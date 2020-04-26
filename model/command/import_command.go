package command

type (
	ImportCommand struct {
		ImportPath string `short:"a" long:"application-path" description:"Path to application folder"  required:"false"`
	}
)
