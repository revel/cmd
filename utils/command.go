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
	c.Env = []string{"GOPATH=" + build.Default.GOPATH, "PATH=" + GetEnv("PATH"), "GOROOT="+ GetEnv("GOROOT")}

}

// Returns an environment variable
func GetEnv(name string) string {
	for _, v := range os.Environ() {
		split := strings.Split(v, "=")
		if split[0] == name {
			return strings.Join(split[1:], "")
		}
	}
	return ""
}
