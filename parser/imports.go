package parser

import (
	"go/ast"
	"go/build"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/revel/cmd/utils"
)

// Add imports to the map from the source dir.
func addImports(imports map[string]string, decl ast.Decl, srcDir string) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return
	}

	if genDecl.Tok != token.IMPORT {
		return
	}

	for _, spec := range genDecl.Specs {
		importSpec := spec.(*ast.ImportSpec)
		var pkgAlias string
		if importSpec.Name != nil {
			pkgAlias = importSpec.Name.Name
			if pkgAlias == "_" {
				continue
			}
		}
		quotedPath := importSpec.Path.Value // e.g. "\"sample/app/models\""
		if quotedPath == `"C"` {
			continue
		}
		fullPath := quotedPath[1 : len(quotedPath)-1] // Remove the quotes

		// If the package was not aliased (common case), we have to import it
		// to see what the package name is.
		// TODO: Can improve performance here a lot:
		// 1. Do not import everything over and over again.  Keep a cache.
		// 2. Exempt the standard library; their directories always match the package name.
		// 3. Can use build.FindOnly and then use parser.ParseDir with mode PackageClauseOnly
		if pkgAlias == "" {
			utils.Logger.Debug("Reading from build", "path", fullPath, "srcPath", srcDir, "gopath", build.Default.GOPATH)
			pkg, err := build.Import(fullPath, srcDir, 0)
			if err != nil {
				// We expect this to happen for apps using reverse routing (since we
				// have not yet generated the routes).  Don't log that.
				if !strings.HasSuffix(fullPath, "/app/routes") {
					utils.Logger.Warn("Could not find import:", "path", fullPath, "srcPath", srcDir, "error", err)
				}
				continue
			} else {
				utils.Logger.Debug("Found package in dir", "dir", pkg.Dir, "name", pkg.ImportPath)
			}
			pkgAlias = pkg.Name
		}

		imports[pkgAlias] = fullPath
	}
}

// Returns a valid import string from the path
// using the build.Defaul.GOPATH to determine the root.
func importPathFromPath(root, basePath string) string {
	vendorTest := filepath.Join(basePath, "vendor")
	if len(root) > len(vendorTest) && root[:len(vendorTest)] == vendorTest {
		return filepath.ToSlash(root[len(vendorTest)+1:])
	}
	for _, gopath := range filepath.SplitList(build.Default.GOPATH) {
		srcPath := filepath.Join(gopath, "src")
		if strings.HasPrefix(root, srcPath) {
			return filepath.ToSlash(root[len(srcPath)+1:])
		}
	}

	srcPath := filepath.Join(build.Default.GOROOT, "src", "pkg")
	if strings.HasPrefix(root, srcPath) {
		utils.Logger.Warn("Code path should be in GOPATH, but is in GOROOT:", "path", root)
		return filepath.ToSlash(root[len(srcPath)+1:])
	}

	utils.Logger.Error("Unexpected! Code path is not in GOPATH:", "path", root)
	return ""
}
