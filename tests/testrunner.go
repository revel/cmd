// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package tests

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"github.com/revel/cmd/utils"
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

var (
	testSuites []TestSuiteDesc // A list of all available tests.

	none = []reflect.Value{} // It is used as input for reflect call in a few places.

	// registeredTests simplifies the search of test suites by their name.
	// "TestSuite.TestName" is used as a key. Value represents index in testSuites.
	registeredTests map[string]int
)

/*
	Below are helper functions.
*/

// describeSuite expects testsuite interface as input parameter
// and returns its description in a form of TestSuiteDesc structure.
func describeSuite(testSuite interface{}) TestSuiteDesc {
	t := reflect.TypeOf(testSuite)

	// Get a list of methods of the embedded test type.
	// It will be used to make sure the same tests are not included in multiple test suites.
	super := t.Elem().Field(0).Type
	superMethods := map[string]bool{}
	for i := 0; i < super.NumMethod(); i++ {
		// Save the current method's name.
		superMethods[super.Method(i).Name] = true
	}

	// Get a list of methods on the test suite that take no parameters, return
	// no results, and were not part of the embedded type's method set.
	var tests []TestDesc
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		mt := m.Type

		// Make sure the test method meets the criterias:
		// - method of testSuite without input parameters;
		// - nothing is returned;
		// - has "Test" prefix;
		// - doesn't belong to the embedded structure.
		methodWithoutParams := (mt.NumIn() == 1 && mt.In(0) == t)
		nothingReturned := (mt.NumOut() == 0)
		hasTestPrefix := (strings.HasPrefix(m.Name, "Test"))
		if methodWithoutParams && nothingReturned && hasTestPrefix && !superMethods[m.Name] {
			// Register the test suite's index so we can quickly find it by test's name later.
			registeredTests[t.Elem().Name()+"."+m.Name] = len(testSuites)

			// Add test to the list of tests.
			tests = append(tests, TestDesc{m.Name})
		}
	}

	return TestSuiteDesc{
		Name:  t.Elem().Name(),
		Tests: tests,
		Elem:  t.Elem(),
	}
}

// errorSummary gets an error and returns its summary in human readable format.
func errorSummary(err *utils.SourceError) (message string) {
	expectedPrefix := "(expected)"
	actualPrefix := "(actual)"
	errDesc := err.Description
	//strip the actual/expected stuff to provide more condensed display.
	if strings.Index(errDesc, expectedPrefix) == 0 {
		errDesc = errDesc[len(expectedPrefix):]
	}
	if strings.LastIndex(errDesc, actualPrefix) > 0 {
		errDesc = errDesc[0 : len(errDesc)-len(actualPrefix)]
	}

	errFile := err.Path
	slashIdx := strings.LastIndex(errFile, "/")
	if slashIdx > 0 {
		errFile = errFile[slashIdx+1:]
	}

	message = fmt.Sprintf("%s %s#%d", errDesc, errFile, err.Line)

	/*
		// If line of error isn't known return the message as is.
		if err.Line == 0 {
			return
		}

		// Otherwise, include info about the line number and the relevant
		// source code lines.
		message += fmt.Sprintf(" (around line %d): ", err.Line)
		for _, line := range err.ContextSource() {
			if line.IsError {
				message += line.Source
			}
		}
	*/

	return
}

//sortbySuiteName sorts the testsuites by name.
type sortBySuiteName []interface{}

func (a sortBySuiteName) Len() int      { return len(a) }
func (a sortBySuiteName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sortBySuiteName) Less(i, j int) bool {
	return reflect.TypeOf(a[i]).Elem().Name() < reflect.TypeOf(a[j]).Elem().Name()
}
