package main_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/revel/cmd/model"
	main "github.com/revel/cmd/revel"
	"github.com/stretchr/testify/assert"
)

// test the commands.
func TestVersion(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-version", a)

	t.Run("Version", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("version-test", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "Check new")
		c.Build.ImportPath = c.ImportPath
		c.Build.TargetPath = filepath.Join(gopath, "build-test", "target")
		a.Nil(main.Commands[model.BUILD].RunWith(c), "Failed to run build")
		c.Index = model.VERSION
		c.Version.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.VERSION].RunWith(c), "Failed to run version-test")
	})
	t.Run("Version-Nobuild", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("version-test2", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "Check new")
		c.Index = model.VERSION
		c.Version.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.VERSION].RunWith(c), "Failed to run version-test")
	})

	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil && !errors.Is(err, os.ErrNotExist) {
			a.Fail("Failed to remove test path", err.Error())
		}
	}
}
