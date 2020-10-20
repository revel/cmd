package utils

import (
	"bytes"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Initialize the command based on the GO environment.
func CmdInit(c *exec.Cmd, addGoPath bool, basePath string) {
	c.Dir = basePath
	// Dep does not like paths that are not real, convert all paths in go to real paths
	realPath := &bytes.Buffer{}
	if addGoPath {
		for _, p := range filepath.SplitList(build.Default.GOPATH) {
			rp, _ := filepath.EvalSymlinks(p)
			if realPath.Len() > 0 {
				realPath.WriteString(string(filepath.ListSeparator))
			}
			realPath.WriteString(rp)
		}
		// Go 1.8 fails if we do not include the GOROOT
		c.Env = []string{"GOPATH=" + realPath.String(), "GOROOT=" + os.Getenv("GOROOT")}
	}
	// Fetch the rest of the env variables
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		if pair[0] == "GOPATH" || pair[0] == "GOROOT" {
			continue
		}
		c.Env = append(c.Env, e)
	}
}
