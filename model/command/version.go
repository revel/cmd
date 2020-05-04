package command
type (
	Version struct {
		ImportCommand
		Update bool `short:"u" long:"update" description:"Update the framework and modules" required:"false"`
		UpdateVersion string `long:"update-version" description:"Specify the version the revel and app will be switched to" required:"false"`
	}
)
