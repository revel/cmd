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
	// Fetch the rest of the env variables
	c.Env = ReducedEnv(addGoPath)

}

// ReducedEnv returns a list of environment vairables by using os.Env
// it will remove the GOPATH, GOROOT if addGoPath is true
func ReducedEnv(addGoPath bool) []string {
	realPath := &bytes.Buffer{}
	env := []string{}
	if addGoPath {
		for _, p := range filepath.SplitList(build.Default.GOPATH) {
			rp, _ := filepath.EvalSymlinks(p)
			if realPath.Len() > 0 {
				realPath.WriteString(string(filepath.ListSeparator))
			}
			realPath.WriteString(rp)
		}
		// Go 1.8 fails if we do not include the GOROOT
		env = []string{"GOPATH=" + realPath.String(), "GOROOT=" + os.Getenv("GOROOT")}
	}

	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		// Always exclude gomodcache
		if pair[0] == "GOMODCACHE" {
			continue
		} else if !addGoPath && (pair[0] == "GOPATH" || pair[0] == "GOROOT") {
			continue
		}
		env = append(env, e)
	}
	return env
}
