// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/revel/cmd/harness"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/tests"
	"github.com/revel/cmd/utils"
)

var cmdTest = &Command{
	UsageLine: "test <import path> [<run mode> <suite.method>]",
	Short:     "run all tests from the command-line",
	Long: `
Run all tests for the Revel app named by the given import path.

For example, to run the booking sample application's tests:

    revel test github.com/revel/examples/booking dev

The run mode is used to select which set of app.conf configuration should
apply and may be used to determine logic in the application itself.

Run mode defaults to "dev".

You can run a specific suite (and function) by specifying a third parameter.
For example, to run all of UserTest:

    revel test outspoken test UserTest

or one of UserTest's methods:

    revel test outspoken test UserTest.Test1
`,
}

func init() {
	cmdTest.RunWith = testApp
	cmdTest.UpdateConfig = updateTestConfig
}

// Called to update the config command with from the older stype.
func updateTestConfig(c *model.CommandConfig, args []string) bool {
	c.Index = model.TEST
	if len(args) == 0 && c.Test.ImportPath != "" {
		return true
	}

	// The full test runs
	// revel test <import path> (run mode) (suite(.function))
	if len(args) < 1 {
		return false
	}
	c.Test.ImportPath = args[0]
	if len(args) > 1 {
		c.Test.Mode = args[1]
	}
	if len(args) > 2 {
		c.Test.Function = args[2]
	}
	return true
}

// Called to test the application.
func testApp(c *model.CommandConfig) (err error) {
	mode := DefaultRunMode
	if c.Test.Mode != "" {
		mode = c.Test.Mode
	}

	// Find and parse app.conf
	revelPath, err := model.NewRevelPaths(mode, c.ImportPath, c.AppPath, model.NewWrappedRevelCallback(nil, c.PackageResolver))
	if err != nil {
		return
	}

	// todo Ensure that the testrunner is loaded in this mode.

	// Create a directory to hold the test result files.
	resultPath := filepath.Join(revelPath.BasePath, "test-results")
	if err = os.RemoveAll(resultPath); err != nil {
		return utils.NewBuildError("Failed to remove test result directory ", "path", resultPath, "error", err)
	}
	if err = os.Mkdir(resultPath, 0777); err != nil {
		return utils.NewBuildError("Failed to create test result directory ", "path", resultPath, "error", err)
	}

	// Direct all the output into a file in the test-results directory.
	file, err := os.OpenFile(filepath.Join(resultPath, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return utils.NewBuildError("Failed to create test result log file: ", "error", err)
	}

	app, reverr := harness.Build(c, revelPath)
	if reverr != nil {
		return utils.NewBuildIfError(reverr, "Error building: ")
	}
	var paths []byte
	if len(app.PackagePathMap) > 0 {
		paths, _ = json.Marshal(app.PackagePathMap)
	}
	runMode := fmt.Sprintf(`{"mode":"%s", "specialUseFlag":%v,"packagePathMap":%s}`, app.Paths.RunMode, c.Verbose[0], string(paths))
	if c.HistoricMode {
		runMode = app.Paths.RunMode
	}
	cmd := app.Cmd(runMode)
	cmd.Dir = c.AppPath

	cmd.Stderr = io.MultiWriter(cmd.Stderr, file)
	cmd.Stdout = io.MultiWriter(cmd.Stderr, file)

	// Start the app...
	if err := cmd.Start(c); err != nil {
		return utils.NewBuildError("Unable to start server", "error", err)
	}
	defer cmd.Kill()

	httpAddr := revelPath.HTTPAddr
	if httpAddr == "" {
		httpAddr = "localhost"
	}

	httpProto := "http"
	if revelPath.HTTPSsl {
		httpProto = "https"
	}

	// Get a list of tests
	baseURL := fmt.Sprintf("%s://%s:%d", httpProto, httpAddr, revelPath.HTTPPort)

	utils.Logger.Infof("Testing %s (%s) in %s mode URL %s \n", revelPath.AppName, revelPath.ImportPath, mode, baseURL)
	testSuites, _ := getTestsList(baseURL)

	// If a specific TestSuite[.Method] is specified, only run that suite/test
	if c.Test.Function != "" {
		testSuites = filterTestSuites(testSuites, c.Test.Function)
	}

	testSuiteCount := len(*testSuites)
	fmt.Printf("\n%d test suite%s to run.\n", testSuiteCount, pluralize(testSuiteCount, "", "s"))
	fmt.Println()

	// Run each suite.
	failedResults, overallSuccess := runTestSuites(revelPath, baseURL, resultPath, testSuites)

	fmt.Println()
	if overallSuccess {
		writeResultFile(resultPath, "result.passed", "passed")
		fmt.Println("All Tests Passed.")
	} else {
		for _, failedResult := range *failedResults {
			fmt.Printf("Failures:\n")
			for _, result := range failedResult.Results {
				if !result.Passed {
					fmt.Printf("%s.%s\n", failedResult.Name, result.Name)
					fmt.Printf("%s\n\n", result.ErrorSummary)
				}
			}
		}
		writeResultFile(resultPath, "result.failed", "failed")
		utils.Logger.Errorf("Some tests failed.  See file://%s for results.", resultPath)
	}

	return
}

// Outputs the results to a file.
func writeResultFile(resultPath, name, content string) {
	if err := ioutil.WriteFile(filepath.Join(resultPath, name), []byte(content), 0666); err != nil {
		utils.Logger.Errorf("Failed to write result file %s: %s", filepath.Join(resultPath, name), err)
	}
}

// Determines if response should be plural.
func pluralize(num int, singular, plural string) string {
	if num == 1 {
		return singular
	}
	return plural
}

// Filters test suites and individual tests to match
// the parsed command line parameter.
func filterTestSuites(suites *[]tests.TestSuiteDesc, suiteArgument string) *[]tests.TestSuiteDesc {
	var suiteName, testName string
	argArray := strings.Split(suiteArgument, ".")
	suiteName = argArray[0]
	if suiteName == "" {
		return suites
	}
	if len(argArray) == 2 {
		testName = argArray[1]
	}
	for _, suite := range *suites {
		if suite.Name != suiteName {
			continue
		}
		if testName == "" {
			return &[]tests.TestSuiteDesc{suite}
		}
		// Only run a particular test in a suite
		for _, test := range suite.Tests {
			if test.Name != testName {
				continue
			}
			return &[]tests.TestSuiteDesc{
				{
					Name:  suite.Name,
					Tests: []tests.TestDesc{test},
				},
			}
		}
		utils.Logger.Errorf("Couldn't find test %s in suite %s", testName, suiteName)
	}
	utils.Logger.Errorf("Couldn't find test suite %s", suiteName)
	return nil
}

// Get a list of tests from server.
// Since this is the first request to the server, retry/sleep a couple times
// in case it hasn't finished starting up yet.
func getTestsList(baseURL string) (*[]tests.TestSuiteDesc, error) {
	var (
		err        error
		resp       *http.Response
		testSuites []tests.TestSuiteDesc
	)
	for i := 0; ; i++ {
		if resp, err = http.Get(baseURL + "/@tests.list"); err == nil {
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if i < 3 {
			time.Sleep(3 * time.Second)
			continue
		}
		if err != nil {
			utils.Logger.Fatalf("Failed to request test list: %s %s", baseURL, err)
		} else {
			utils.Logger.Fatalf("Failed to request test list: non-200 response %s", baseURL)
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	err = json.NewDecoder(resp.Body).Decode(&testSuites)

	return &testSuites, err
}

// Run the testsuites using the container.
func runTestSuites(paths *model.RevelContainer, baseURL, resultPath string, testSuites *[]tests.TestSuiteDesc) (*[]tests.TestSuiteResult, bool) {
	// We can determine the testsuite location by finding the test module and extracting the data from it
	resultFilePath := filepath.Join(paths.ModulePathMap["testrunner"].Path, "app", "views", "TestRunner/SuiteResult.html")

	var (
		overallSuccess = true
		failedResults  []tests.TestSuiteResult
	)
	for _, suite := range *testSuites {
		// Print the name of the suite we're running.
		name := suite.Name
		if len(name) > 22 {
			name = name[:19] + "..."
		}
		fmt.Printf("%-22s", name)

		// Run every test.
		startTime := time.Now()
		suiteResult := tests.TestSuiteResult{Name: suite.Name, Passed: true}
		for _, test := range suite.Tests {
			testURL := baseURL + "/@tests/" + suite.Name + "/" + test.Name
			resp, err := http.Get(testURL)
			if err != nil {
				utils.Logger.Errorf("Failed to fetch test result at url %s: %s", testURL, err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			var testResult tests.TestResult
			err = json.NewDecoder(resp.Body).Decode(&testResult)
			if err == nil && !testResult.Passed {
				suiteResult.Passed = false
				utils.Logger.Error("Test Failed", "suite", suite.Name, "test", test.Name)
				fmt.Printf("   %s.%s : FAILED\n", suite.Name, test.Name)
			} else {
				fmt.Printf("   %s.%s : PASSED\n", suite.Name, test.Name)
			}
			suiteResult.Results = append(suiteResult.Results, testResult)
		}
		overallSuccess = overallSuccess && suiteResult.Passed

		// Print result.  (Just PASSED or FAILED, and the time taken)
		suiteResultStr, suiteAlert := "PASSED", ""
		if !suiteResult.Passed {
			suiteResultStr, suiteAlert = "FAILED", "!"
			failedResults = append(failedResults, suiteResult)
		}
		fmt.Printf("%8s%3s%6ds\n", suiteResultStr, suiteAlert, int(time.Since(startTime).Seconds()))
		// Create the result HTML file.
		suiteResultFilename := filepath.Join(resultPath,
			fmt.Sprintf("%s.%s.html", suite.Name, strings.ToLower(suiteResultStr)))
		if err := utils.RenderTemplate(suiteResultFilename, resultFilePath, suiteResult); err != nil {
			utils.Logger.Error("Failed to render template", "error", err)
		}
	}

	return &failedResults, overallSuccess
}
