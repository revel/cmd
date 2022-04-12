package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

// DirExists returns true if the given path exists and is a directory.
func DirExists(filename string) bool {
	fileInfo, err := os.Stat(filename)
	return err == nil && fileInfo.IsDir()
}

// MustReadLines reads the lines of the given file.  Panics in the case of error.
func MustReadLines(filename string) []string {
	r, err := ReadLines(filename)
	if err != nil {
		panic(err)
	}
	return r
}

// ReadLines reads the lines of the given file.  Panics in the case of error.
func ReadLines(filename string) ([]string, error) {
	dataBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(dataBytes), "\n"), nil
}

// Copy file returns error.
func CopyFile(destFilename, srcFilename string) (err error) {
	destFile, err := os.Create(destFilename)
	if err != nil {
		return NewBuildIfError(err, "Failed to create file", "file", destFilename)
	}

	srcFile, err := os.Open(srcFilename)
	if err != nil {
		return NewBuildIfError(err, "Failed to open file", "file", srcFilename)
	}

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return NewBuildIfError(err, "Failed to copy data", "fromfile", srcFilename, "tofile", destFilename)
	}

	err = destFile.Close()
	if err != nil {
		return NewBuildIfError(err, "Failed to close file", "file", destFilename)
	}

	err = srcFile.Close()
	if err != nil {
		return NewBuildIfError(err, "Failed to close file", "file", srcFilename)
	}

	return
}

// GenerateTemplate renders the given template to produce source code, which it writes
// to the given file.
func GenerateTemplate(filename, templateSource string, args map[string]interface{}) (err error) {
	tmpl := template.Must(template.New("").Parse(templateSource))

	var b bytes.Buffer
	if err = tmpl.Execute(&b, args); err != nil {
		return NewBuildIfError(err, "ExecuteTemplate: Execute failed")
	}
	sourceCode := b.String()
	filePath := filepath.Dir(filename)
	if !DirExists(filePath) {
		err = os.MkdirAll(filePath, 0777)
		if err != nil && !os.IsExist(err) {
			return NewBuildIfError(err, "Failed to make directory", "dir", filePath)
		}
	}

	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		Logger.Fatal("Failed to create file", "error", err)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err = file.WriteString(sourceCode); err != nil {
		Logger.Fatal("Failed to write to file: ", "error", err)
	}

	return
}

// Given the target path and source path and data. A template.
func RenderTemplate(destPath, srcPath string, data interface{}) (err error) {
	tmpl, err := template.ParseFiles(srcPath)
	if err != nil {
		return NewBuildIfError(err, "Failed to parse template "+srcPath)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return NewBuildIfError(err, "Failed to create  ", "path", destPath)
	}

	err = tmpl.Execute(f, data)
	if err != nil {
		return NewBuildIfError(err, "Failed to Render template "+srcPath)
	}

	err = f.Close()
	if err != nil {
		return NewBuildIfError(err, "Failed to close file stream "+destPath)
	}
	return
}

// Given the target path and source path and data. A template.
func RenderTemplateToStream(output io.Writer, srcPath []string, data interface{}) (err error) {
	tmpl, err := template.ParseFiles(srcPath...)
	if err != nil {
		return NewBuildIfError(err, "Failed to parse template "+srcPath[0])
	}

	err = tmpl.Execute(output, data)
	if err != nil {
		return NewBuildIfError(err, "Failed to render template "+srcPath[0])
	}
	return
}

func MustChmod(filename string, mode os.FileMode) {
	err := os.Chmod(filename, mode)
	PanicOnError(err, fmt.Sprintf("Failed to chmod %d %q", mode, filename))
}

// Called if panic.
func PanicOnError(err error, msg string) {
	var serr *SourceError
	if (errors.As(err, &serr) && serr != nil) || err != nil {
		Logger.Panicf("Abort: %s: %s %s", msg, serr, err)
	}
}

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".template" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".template" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func CopyDir(destDir, srcDir string, data map[string]interface{}) error {
	if !DirExists(srcDir) {
		return nil
	}
	return fsWalk(srcDir, srcDir, func(srcPath string, info os.FileInfo, err error) error {
		// Get the relative path from the source base, and the corresponding path in
		// the dest directory.
		relSrcPath := strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator))
		destPath := filepath.Join(destDir, relSrcPath)

		// Skip dot files and dot directories.
		if strings.HasPrefix(relSrcPath, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create a subdirectory if necessary.
		if info.IsDir() {
			err := os.MkdirAll(filepath.Join(destDir, relSrcPath), 0777)
			if !os.IsExist(err) {
				return NewBuildIfError(err, "Failed to create directory", "path", destDir+"/"+relSrcPath)
			}
			return nil
		}

		// If this file ends in ".template", render it as a template.
		if strings.HasSuffix(relSrcPath, ".template") {
			return RenderTemplate(destPath[:len(destPath)-len(".template")], srcPath, data)
		}

		// Else, just copy it over.

		return CopyFile(destPath, srcPath)
	})
}

// Shortcut to fsWalk.
func Walk(root string, walkFn filepath.WalkFunc) error {
	return fsWalk(root, root, walkFn)
}

// Walk the path tree using the function
// Every file found will call the function.
func fsWalk(fname string, linkName string, walkFn filepath.WalkFunc) error {
	fsWalkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var name string
		name, err = filepath.Rel(fname, path)
		if err != nil {
			return err
		}

		path = filepath.Join(linkName, name)

		if err == nil && info.Mode()&os.ModeSymlink == os.ModeSymlink {
			var symlinkPath string
			symlinkPath, err = filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}

			// https://github.com/golang/go/blob/master/src/path/filepath/path.go#L392
			info, err = os.Lstat(symlinkPath)

			if err != nil {
				return walkFn(path, info, err)
			}

			if info.IsDir() {
				return fsWalk(symlinkPath, path, walkFn)
			}
		}

		return walkFn(path, info, err)
	}
	err := filepath.Walk(fname, fsWalkFunc)
	return err
}

// Tar gz the folder.
func TarGzDir(destFilename, srcDir string) (name string, err error) {
	zipFile, err := os.Create(destFilename)
	if err != nil {
		return "", NewBuildIfError(err, "Failed to create archive", "file", destFilename)
	}

	defer func() {
		_ = zipFile.Close()
	}()

	gzipWriter := gzip.NewWriter(zipFile)
	defer func() {
		_ = gzipWriter.Close()
	}()

	tarWriter := tar.NewWriter(gzipWriter)
	defer func() {
		_ = tarWriter.Close()
	}()

	err = fsWalk(srcDir, srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			Logger.Debugf("error in walkFn: %s", err)
		}

		if info.IsDir() {
			return nil
		}

		srcFile, err := os.Open(srcPath)
		if err != nil {
			return NewBuildIfError(err, "Failed to read file", "file", srcPath)
		}

		defer func() {
			_ = srcFile.Close()
		}()

		err = tarWriter.WriteHeader(&tar.Header{
			Name:    strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator)),
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		})
		if err != nil {
			return NewBuildIfError(err, "Failed to write tar entry header", "file", srcPath)
		}

		_, err = io.Copy(tarWriter, srcFile)
		if err != nil {
			return NewBuildIfError(err, "Failed to copy file", "file", srcPath)
		}

		return nil
	})

	return zipFile.Name(), err
}

// Return true if the file exists.
func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// empty returns true if the given directory is empty.
// the directory must exist.
func Empty(dirname string) bool {
	if !DirExists(dirname) {
		return true
	}
	dir, err := os.Open(dirname)
	if err != nil {
		Logger.Infof("error opening directory: %s", err)
		return false
	}
	defer func() {
		_ = dir.Close()
	}()
	results, _ := dir.Readdir(1)
	return len(results) == 0
}

// Find the full source dir for the import path, uses the build.Default.GOPATH to search for the directory.
func FindSrcPaths(appPath string, packageList []string, packageResolver func(pkgName string) error) (sourcePathsmap map[string]string, err error) {
	sourcePathsmap, missingList, err := findSrcPaths(appPath, packageList)
	if err != nil && packageResolver != nil || len(missingList) > 0 {
		Logger.Info("Failed to find package, attempting to call resolver for missing packages", "missing packages", missingList)
		for _, item := range missingList {
			if err = packageResolver(item); err != nil {
				return
			}
		}
		sourcePathsmap, missingList, err = findSrcPaths(appPath, packageList)
	}
	if err != nil && len(missingList) > 0 {
		for _, missing := range missingList {
			Logger.Error("Unable to import this package", "package", missing)
		}
	}

	return
}

// Error is used for constant errors.
type Error string

// Error implements the error interface.
func (e Error) Error() string {
	return string(e)
}

var (
	ErrNoApp   Error = "no app found"
	ErrNoRevel Error = "no revel found"
)

// Find the full source dir for the import path, uses the build.Default.GOPATH to search for the directory.
func findSrcPaths(appPath string, packagesList []string) (sourcePathsmap map[string]string, missingList []string, err error) {
	// Use packages to fetch
	// by not specifying env, we will use the default env
	config := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedDeps,
		Dir:  appPath,
	}
	config.Env = ReducedEnv(false)
	sourcePathsmap = map[string]string{}
	Logger.Infof("Environment path %s root %s config env %s", os.Getenv("GOPATH"), os.Getenv("GOROOT"), config.Env)

	pkgs, err := packages.Load(config, packagesList...)
	Logger.Infof("Environment path %s root %s config env %s", os.Getenv("GOPATH"), os.Getenv("GOROOT"), config.Env)
	Logger.Info("Loaded packages ", "len results", len(pkgs), "error", err, "basedir", appPath)
	for _, packageName := range packagesList {
		found := false
		log := Logger.New("seeking", packageName)
		for _, pck := range pkgs {
			log.Info("Found package", "package", pck.ID)
			if pck.ID == packageName {
				if pck.Errors != nil && len(pck.Errors) > 0 {
					log.Error("Error ", "count", len(pck.Errors), "App Import Path", pck.ID, "filesystem path", pck.PkgPath, "errors", pck.Errors)
					// continue
				}
				// a,_ := pck.MarshalJSON()
				log.Info("Found ", "count", len(pck.GoFiles), "App Import Path", pck.ID, "apppath", appPath)
				if len(pck.GoFiles) > 0 {
					sourcePathsmap[packageName] = filepath.Dir(pck.GoFiles[0])
					found = true
				}
			}
		}
		if !found {
			if packageName == "github.com/revel/revel" {
				err = ErrNoRevel
			} else {
				err = ErrNoApp
			}
			missingList = append(missingList, packageName)
		}
	}

	return
}
