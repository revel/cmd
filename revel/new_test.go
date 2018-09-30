package main_test

import (
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/revel"
	"github.com/revel/cmd/utils"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// test the commands
func TestNew(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-new",  a)

		t.Run("New", func(t *testing.T) {
			a := assert.New(t)
			c := newApp("new-test", model.NEW, nil, a)
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
			c.New.SkeletonPath = "git://github.com/revel/cmd:skeleton2"
			a.NotNil(main.Commands[model.NEW].RunWith(c), "Expected Failed to run with new")
			// We need to pick a different path
			c = newApp("new/test/c/2", model.NEW, nil, a)
			c.New.SkeletonPath = "git://github.com/revel/cmd:skeleton"
			a.Nil(main.Commands[model.NEW].RunWith(c), "Failed to run with new skeleton git")
		})
		t.Run("Skeleton-Go", func(t *testing.T) {
			a := assert.New(t)
			c := newApp("new/test/d", model.NEW, nil, a)
			c.New.SkeletonPath = "github.com/revel/cmd:skeleton"
			a.Nil(main.Commands[model.NEW].RunWith(c), "Failed to run with new from go")
		})
	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}

// test the commands
func TestNewVendor(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-new-vendor",  a)
	precall := func(c *model.CommandConfig) {
		c.New.Vendored = true
	}
	t.Run("New", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("onlyone/v/a", model.NEW, precall, a)
		c.New.Vendored = true
		a.Nil(main.Commands[model.NEW].RunWith(c), "New failed")
	})
	t.Run("Test", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("onlyone/v/a", model.TEST, nil, a)
		a.Nil(main.Commands[model.TEST].RunWith(c), "Test failed")
	})
	t.Run("Build", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("onlyone/v/a", model.BUILD, nil, a)
		c.Index = model.BUILD
		c.Build.TargetPath = filepath.Join(gopath, "src/onlyone/v/a", "target")
		a.Nil(main.Commands[model.BUILD].RunWith(c), " Build failed")
		a.True(utils.DirExists(c.Build.TargetPath), "Target folder not made", c.Build.TargetPath)
	})
	t.Run("Package", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("onlyone/v/a", model.PACKAGE, nil, a)
		c.Package.TargetPath = filepath.Join(gopath, "src/onlyone/v/a", "target.tar.gz")
		a.Nil(main.Commands[model.PACKAGE].RunWith(c), "Package Failed")
		a.True(utils.Exists(c.Package.TargetPath), "Target package not made", c.Package.TargetPath)
	})
	t.Run("TestVendorDir", func(t *testing.T) {
		// Check to see that no additional packages were downloaded outside the vendor folder
		files, err := ioutil.ReadDir(gopath)
		a.Nil(err, "Failed to read gopath folder")
		// bin/     onlyone/ pkg/     src/
		a.Equal(3, len(files), "Expected single file in "+gopath)
		files, err = ioutil.ReadDir(filepath.Join(gopath, "src"))
		a.Nil(err, "Failed to read src folder")
		a.Equal(1, len(files), "Expected single file in source folder", filepath.Join(gopath, "src"))
	})
	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
