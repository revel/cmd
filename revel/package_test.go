package main_test

import (
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/revel"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// test the commands
func TestPackage(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-package",  a)

	t.Run("Package", func(t *testing.T) {
		a := assert.New(t)
		c := newApp("package-test", model.NEW, nil, a)
		main.Commands[model.NEW].RunWith(c)
		c.Index = model.PACKAGE
		c.Package.ImportPath = c.ImportPath
		a.Nil(main.Commands[model.PACKAGE].RunWith(c), "Failed to run package-test")
	})

	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
