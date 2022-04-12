package main_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/revel/cmd/model"
	main "github.com/revel/cmd/revel"
	"github.com/revel/cmd/utils"
	"github.com/stretchr/testify/assert"
)

// test the commands.
func TestBuild(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-build", a)

	t.Run("Build", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("build-test", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "failed to run new")
		c.Index = model.BUILD
		c.Build.TargetPath = filepath.Join(gopath, "build-test", "target")
		c.Build.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.BUILD].RunWith(c), "Failed to run build-test")
		a.True(utils.Exists(filepath.Join(gopath, "build-test", "target")))
	})

	t.Run("Build-withFlags", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("build-test-WithFlags", model.NEW, nil, a)
		c.BuildFlags = []string{
			"build-test-WithFlags/app.AppVersion=SomeValue",
			"build-test-WithFlags/app.SomeOtherValue=SomeValue",
		}
		a.Nil(main.Commands[model.NEW].RunWith(c), "failed to run new")
		c.Index = model.BUILD
		c.Build.TargetPath = filepath.Join(gopath, "build-test", "target")
		c.Build.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.BUILD].RunWith(c), "Failed to run build-test-withFlags")
		a.True(utils.Exists(filepath.Join(gopath, "build-test", "target")))
	})

	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
