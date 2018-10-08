package utils

import (
	"go/build"
	"os"
	"os/exec"
	"strings"
)

// Initialize the command based on the GO environment
func CmdInit(c *exec.Cmd, basePath string) {
	c.Dir = basePath
	// Go 1.8 fails if we do not include the GOROOT
	c.Env = []string{"GOPATH=" + build.Default.GOPATH, "GOROOT="+ os.Getenv("GOROOT")}
	// Fetch the rest of the env variables
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0]=="GOPATH" || pair[0]=="GOROOT" {
			continue
		}
		c.Env = append(c.Env,e)
	}
}