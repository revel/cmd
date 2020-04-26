package command
type (
	Version struct {
		ImportCommand
		Update bool `short:"u" long:"Update the framework and modules" required:"false"`
	}
)
