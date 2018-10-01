package main_test

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// test the commands
func TestRun(t *testing.T) {
	a := assert.New(t)
	gopath := setup("revel-test-run",  a)

	// TODO Testing run

	if !t.Failed() {
		if err := os.RemoveAll(gopath); err != nil {
			a.Fail("Failed to remove test path")
		}
	}
}
