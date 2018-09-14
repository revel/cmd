// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"

	"github.com/revel/cmd/model"
	"go/build"
	"go/token"
	"go/parser"
	"go/ast"
	"io/ioutil"
	"path/filepath"
	"github.com/revel/cmd/utils"
)

var cmdVersion = &Command{
	UsageLine: "version",
	Short:     "displays the Revel Framework and Go version",
	Long: `
Displays the Revel Framework and Go version.

For example:

    revel version
`,
}

func init() {
	cmdVersion.RunWith = versionApp
}

// Displays the version of go and Revel
func versionApp(c *model.CommandConfig) {
	revelPkg, err := build.Import(model.RevelImportPath, c.Version.ImportPath, build.FindOnly)
	if err != nil {
		utils.Logger.Errorf("Failed to find Revel with error:", "error", err)
	}

	utils.Logger.Info("Fullpath to revel", revelPkg.Dir)
	fset := token.NewFileSet() // positions are relative to fset

	version, err := ioutil.ReadFile(filepath.Join(revelPkg.Dir,"version.go"))
	if err != nil {
		utils.Logger.Errorf("Failed to find Revel version:", "error", err)
	}

	// Parse src but stop after processing the imports.
	f, err := parser.ParseFile(fset, "", version, parser.ParseComments)
	if err != nil {
		utils.Logger.Errorf("Failed to parse Revel version error:", "error", err)
	}

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
			fmt.Printf("Revel %s = %s\n",spec.Names[0].Name,r.Value)
		}
	}

	fmt.Printf("\n   %s %s/%s\n\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
