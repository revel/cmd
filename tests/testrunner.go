// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package tests

import (
	"html/template"
	"reflect"
)

// TestSuiteDesc is used for storing information about a single test suite.
// This structure is required by revel test cmd.
type TestSuiteDesc struct {
	Name  string
	Tests []TestDesc

	// Elem is reflect.Type which can be used for accessing methods
	// of the test suite.
	Elem reflect.Type
}

// TestDesc is used for describing a single test of some test suite.
// This structure is required by revel test cmd.
type TestDesc struct {
	Name string
}

// TestSuiteResult stores the results the whole test suite.
// This structure is required by revel test cmd.
type TestSuiteResult struct {
	Name    string
	Passed  bool
	Results []TestResult
}

// TestResult represents the results of running a single test of some test suite.
// This structure is required by revel test cmd.
type TestResult struct {
	Name         string
	Passed       bool
	ErrorHTML    template.HTML
	ErrorSummary string
}
