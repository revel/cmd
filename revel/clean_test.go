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
func TestClean(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-clean", a)

	t.Run("Clean", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("clean-test", model.NEW, nil, a)

		a.Nil(main.Commands[model.NEW].RunWith(c), "failed to run new")

		c.Index = model.TEST
		a.Nil(main.Commands[model.TEST].RunWith(c), "failed to run test")

		a.True(utils.Exists(filepath.Join(gopath, "clean-test", "app", "tmp", "main.go")),
			"Missing main from path "+filepath.Join(gopath, "clean-test", "app", "tmp", "main.go"))
		c.Clean.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.CLEAN].RunWith(c), "Failed to run clean-test")
		a.False(utils.Exists(filepath.Join(gopath, "clean-test", "app", "tmp", "main.go")),
			"Did not remove main from path "+filepath.Join(gopath, "clean-test", "app", "tmp", "main.go"))
	})
	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
