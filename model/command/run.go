package command

type (
	Run struct {
		ImportCommand
		Mode    string `short:"m" long:"run-mode" description:"The mode to run the application in"`
		Port    int    `short:"p" long:"port" default:"-1" description:"The port to listen" `
		NoProxy bool   `short:"n" long:"no-proxy" description:"True if proxy server should not be started. This will only update the main and routes files on change"`
	}
)
