package main_test

import (
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/revel"
	"github.com/revel/cmd/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

// test the commands
func TestBuild(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-build",  a)

	t.Run("Build", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("build-test", model.NEW, nil, a)
		main.Commands[model.NEW].RunWith(c)
		c.Index = model.BUILD
		c.Build.TargetPath = filepath.Join(gopath, "build-test", "target")
		c.Build.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.BUILD].RunWith(c), "Failed to run build-test")
		a.True(utils.Exists(filepath.Join(gopath, "build-test", "target")))
	})

	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
