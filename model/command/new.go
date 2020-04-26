package command
type (
	New struct {
		ImportCommand
		SkeletonPath string `short:"s" long:"skeleton" description:"Path to skeleton folder (Must exist on GO PATH)" required:"false"`
		Package      string `short:"p" long:"package" description:"The package name, this becomes the repfix to the app name, if defined vendored is set to true" required:"false"`
		NotVendored  bool   `short:"V" long:"vendor" description:"True if project should not be configured with a go.mod"`
		Run          bool   `short:"r" long:"run" description:"True if you want to run the application right away"`
	}

)