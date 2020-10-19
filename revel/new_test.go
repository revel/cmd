package main_test

import (
	"os"
	"testing"

	"github.com/revel/cmd/model"
	main "github.com/revel/cmd/revel"
	"github.com/stretchr/testify/assert"
)

// test the commands.
func TestNew(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-new", a)

	t.Run("New", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("new-test", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "New failed")
	})
	t.Run("New-NotVendoredmode", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("new-notvendored", model.NEW, nil, a)
		c.New.NotVendored = true
		a.Nil(main.Commands[model.NEW].RunWith(c), "New failed")
	})
	t.Run("Path", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("new/test/a", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "New path failed")
	})
	t.Run("Path-Duplicate", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("new/test/b", model.NEW, nil, a)
		a.Nil(main.Commands[model.NEW].RunWith(c), "New path failed")
		c = newApp("new/test/b", model.NEW, nil, a)
		a.NotNil(main.Commands[model.NEW].RunWith(c), "Duplicate path Did Not failed")
	})
	t.Run("Skeleton-Git", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("new/test/c/1", model.NEW, nil, a)
		c.New.SkeletonPath = "git://github.com/revel/skeletons:basicnsadnsak"
		a.NotNil(main.Commands[model.NEW].RunWith(c), "Expected Failed to run with new")
		// We need to pick a different path
		c = newApp("new/test/c/2", model.NEW, nil, a)
		c.New.SkeletonPath = "git://github.com/revel/skeletons:basic/bootstrap4"
		a.Nil(main.Commands[model.NEW].RunWith(c), "Failed to run with new skeleton git")
	})
	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
