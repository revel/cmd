package command

type (
	Test struct {
		ImportCommand
		Mode       string `short:"m" long:"run-mode" description:"The mode to run the application in"`
		Function   string `short:"f" long:"suite-function" description:"The suite.function"`
	}
)