// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/revel/cmd"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

type (
	// The version container.
	VersionCommand struct {
		Command        *model.CommandConfig // The command
		revelVersion   *model.Version       // The Revel framework version
		modulesVersion *model.Version       // The Revel modules version
		cmdVersion     *model.Version       // The tool version
	}
)

var cmdVersion = &Command{
	UsageLine: "revel version",
	Short:     "displays the Revel Framework and Go version",
	Long: `
Displays the Revel Framework and Go version.

For example:

    revel version [<application path>]
`,
}

func init() {
	v := &VersionCommand{}
	cmdVersion.UpdateConfig = v.UpdateConfig
	cmdVersion.RunWith = v.RunWith
}

// Update the version.
func (v *VersionCommand) UpdateConfig(c *model.CommandConfig, args []string) bool {
	if len(args) > 0 {
		c.Version.ImportPath = args[0]
	}
	return true
}

// Displays the version of go and Revel.
func (v *VersionCommand) RunWith(c *model.CommandConfig) (err error) {
	utils.Logger.Info("Requesting version information", "config", c)
	v.Command = c

	// Update the versions with the local values
	v.updateLocalVersions()

	needsUpdates := true
	versionInfo := ""
	for x := 0; x < 2 && needsUpdates; x++ {
		versionInfo, needsUpdates = v.doRepoCheck(x == 0)
	}

	fmt.Printf("%s\n\nGo Location:%s\n\n", versionInfo, c.GoCmd)
	cmd := exec.Command(c.GoCmd, "version")
	cmd.Stdout = os.Stdout
	if e := cmd.Start(); e != nil {
		fmt.Println("Go command error ", e)
	} else {
		if err = cmd.Wait(); err != nil {
			return
		}
	}

	return
}

// Checks the Revel repos for the latest version.
func (v *VersionCommand) doRepoCheck(updateLibs bool) (versionInfo string, needsUpdate bool) {
	var (
		title        string
		localVersion *model.Version
	)
	for _, repo := range []string{"revel", "cmd", "modules"} {
		versonFromRepo, err := v.versionFromRepo(repo, "", "version.go")
		if err != nil {
			utils.Logger.Info("Failed to get version from repo", "repo", repo, "error", err)
		}
		switch repo {
		case "revel":
			title, repo, localVersion = "Revel Framework", "github.com/revel/revel", v.revelVersion
		case "cmd":
			title, repo, localVersion = "Revel Cmd", "github.com/revel/cmd/revel", v.cmdVersion
		case "modules":
			title, repo, localVersion = "Revel Modules", "github.com/revel/modules", v.modulesVersion
		}

		// Only do an update on the first loop, and if specified to update
		versionInfo += v.outputVersion(title, repo, localVersion, versonFromRepo)
	}
	return
}

// Prints out the local and remote versions, calls update if needed.
func (v *VersionCommand) outputVersion(title, repo string, local, remote *model.Version) (output string) {
	buffer := &bytes.Buffer{}
	remoteVersion := "Unknown"
	if remote != nil {
		remoteVersion = remote.VersionString()
	}
	localVersion := "Unknown"
	if local != nil {
		localVersion = local.VersionString()
	}

	fmt.Fprintf(buffer, "%s\t:\t%s\t(%s remote master branch)\n", title, localVersion, remoteVersion)
	return buffer.String()
}

// Returns the version from the repository.
func (v *VersionCommand) versionFromRepo(repoName, branchName, fileName string) (version *model.Version, err error) {
	if branchName == "" {
		branchName = "master"
	}
	// Try to download the version of file from the repo, just use an http connection to retrieve the source
	// Assuming that the repo is github
	fullurl := "https://raw.githubusercontent.com/revel/" + repoName + "/" + branchName + "/" + fileName
	resp, err := http.Get(fullurl)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	utils.Logger.Info("Got version file", "from", fullurl, "content", string(body))

	return v.versionFromBytes(body)
}

func (v *VersionCommand) versionFromFilepath(sourcePath string) (version *model.Version, err error) {
	utils.Logger.Info("Fullpath to revel", "dir", sourcePath)

	sourceStream, err := ioutil.ReadFile(filepath.Join(sourcePath, "version.go"))
	if err != nil {
		return
	}
	return v.versionFromBytes(sourceStream)
}

// Returns version information from a file called version on the gopath.
func (v *VersionCommand) versionFromBytes(sourceStream []byte) (version *model.Version, err error) {
	fset := token.NewFileSet() // positions are relative to fset

	// Parse src but stop after processing the imports.
	f, err := parser.ParseFile(fset, "", sourceStream, parser.ParseComments)
	if err != nil {
		err = utils.NewBuildError("Failed to parse Revel version error:", "error", err)
		return
	}
	version = &model.Version{}

	// Print the imports from the file's AST.
	for _, s := range f.Decls {
		genDecl, ok := s.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok != token.CONST {
			continue
		}
		for _, a := range genDecl.Specs {
			spec := a.(*ast.ValueSpec)
			r := spec.Values[0].(*ast.BasicLit)
			switch spec.Names[0].Name {
			case "Version":
				if err = version.ParseVersion(strings.ReplaceAll(r.Value, `"`, "")); err != nil {
					return
				}
			case "BuildDate":
				version.BuildDate = r.Value
			case "MinimumGoVersion":
				version.MinGoVersion = r.Value
			}
		}
	}
	return
}

// Fetch the local version of revel from the file system.
func (v *VersionCommand) updateLocalVersions() {
	v.cmdVersion = &model.Version{}

	if err := v.cmdVersion.ParseVersion(cmd.Version); err != nil {
		utils.Logger.Warn("Error parsing version", "error", err, "version", cmd.Version)
		return
	}

	v.cmdVersion.BuildDate = cmd.BuildDate
	v.cmdVersion.MinGoVersion = cmd.MinimumGoVersion

	if v.Command.Version.ImportPath == "" {
		return
	}

	pathMap, err := utils.FindSrcPaths(v.Command.AppPath, []string{model.RevelImportPath, model.RevelModulesImportPath}, v.Command.PackageResolver)
	if err != nil {
		utils.Logger.Warn("Unable to extract version information from Revel library", "path", pathMap[model.RevelImportPath], "error", err)
		return
	}
	utils.Logger.Info("Fullpath to revel modules", "dir", pathMap[model.RevelImportPath])
	v.revelVersion, err = v.versionFromFilepath(pathMap[model.RevelImportPath])
	if err != nil {
		utils.Logger.Warn("Unable to extract version information from Revel", "error,err")
	}

	v.modulesVersion, err = v.versionFromFilepath(pathMap[model.RevelModulesImportPath])
	if err != nil {
		utils.Logger.Warn("Unable to extract version information from Revel Modules", "path", pathMap[model.RevelModulesImportPath], "error", err)
	}
}
