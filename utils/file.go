package utils


// DirExists returns true if the given path exists and is a directory.
import (
	"os"
	"archive/tar"
	"strings"
	"io"
	"path/filepath"
	"fmt"
	"html/template"
	"compress/gzip"
	"go/build"
	"io/ioutil"
	"bytes"
)

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

func MustCopyFile(destFilename, srcFilename string) {
	destFile, err := os.Create(destFilename)
	PanicOnError(err, "Failed to create file "+destFilename)

	srcFile, err := os.Open(srcFilename)
	PanicOnError(err, "Failed to open file "+srcFilename)

	_, err = io.Copy(destFile, srcFile)
	PanicOnError(err,
		fmt.Sprintf("Failed to copy data from %s to %s", srcFile.Name(), destFile.Name()))

	err = destFile.Close()
	PanicOnError(err, "Failed to close file "+destFile.Name())

	err = srcFile.Close()
	PanicOnError(err, "Failed to close file "+srcFile.Name())
}


// GenerateTemplate renders the given template to produce source code, which it writes
// to the given file.
func MustGenerateTemplate(filename, templateSource string, args map[string]interface{}) (err error) {
	tmpl := template.Must(template.New("").Parse(templateSource))

	var b bytes.Buffer
	if err = tmpl.Execute(&b, args); err != nil {
		Logger.Fatal("ExecuteTemplate: Execute failed", "error", err)
		return
	}
	sourceCode := b.String()
	filePath := filepath.Dir(filename)
	if !DirExists(filePath) {
		err = os.Mkdir(filePath, 0777)
		if err != nil && !os.IsExist(err) {
			Logger.Fatal("Failed to make directory","dir", filePath, "error", err)
		}
	}


	// Create the file
	file, err := os.Create(filename)
	if err != nil {
		Logger.Fatal("Failed to create file","error", err)
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

// Given the target path and source path and data. A template
func MustRenderTemplate(destPath, srcPath string, data interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	PanicOnError(err, "Failed to parse template "+srcPath)

	f, err := os.Create(destPath)
	PanicOnError(err, "Failed to create "+destPath)

	err = tmpl.Execute(f, data)
	PanicOnError(err, "Failed to render template "+srcPath)

	err = f.Close()
	PanicOnError(err, "Failed to close "+f.Name())
}

// Given the target path and source path and data. A template
func MustRenderTemplateToStream(output io.Writer, srcPath []string, data interface{}) {
	tmpl, err := template.ParseFiles(srcPath...)
	PanicOnError(err, "Failed to parse template "+srcPath[0])

	err = tmpl.Execute(output, data)
	PanicOnError(err, "Failed to render template "+srcPath[0])
}

func MustChmod(filename string, mode os.FileMode) {
	err := os.Chmod(filename, mode)
	PanicOnError(err, fmt.Sprintf("Failed to chmod %d %q", mode, filename))
}

// Called if panic
func PanicOnError(err error, msg string) {
	if revErr, ok := err.(*Error); (ok && revErr != nil) || (!ok && err != nil) {
		Logger.Fatalf("Abort: %s: %s %s\n", msg, revErr, err)
		//panic(NewLoggedError(err))
	}
}

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".template" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".template" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func MustCopyDir(destDir, srcDir string, data map[string]interface{}) error {
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
				PanicOnError(err, "Failed to create directory")
			}
			return nil
		}

		// If this file ends in ".template", render it as a template.
		if strings.HasSuffix(relSrcPath, ".template") {
			MustRenderTemplate(destPath[:len(destPath)-len(".template")], srcPath, data)
			return nil
		}

		// Else, just copy it over.
		MustCopyFile(destPath, srcPath)
		return nil
	})
}

func Walk(root string, walkFn filepath.WalkFunc) error {
	return fsWalk(root,root,walkFn)
}
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

func MustTarGzDir(destFilename, srcDir string) string {
	zipFile, err := os.Create(destFilename)
	PanicOnError(err, "Failed to create archive")
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

	_ = fsWalk(srcDir, srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		srcFile, err := os.Open(srcPath)
		PanicOnError(err, "Failed to read source file")
		defer func() {
			_ = srcFile.Close()
		}()

		err = tarWriter.WriteHeader(&tar.Header{
			Name:    strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator)),
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		})
		PanicOnError(err, "Failed to write tar entry header")

		_, err = io.Copy(tarWriter, srcFile)
		PanicOnError(err, "Failed to copy")

		return nil
	})

	return zipFile.Name()
}

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// empty returns true if the given directory is empty.
// the directory must exist.
func Empty(dirname string) bool {
	dir, err := os.Open(dirname)
	if err != nil {
		Logger.Infof("error opening directory: %s", err)
	}
	defer func() {
		_ = dir.Close()
	}()
	results, _ := dir.Readdir(1)
	return len(results) == 0
}

func ImportPathFromCurrentDir() string {
	pwd, _ := os.Getwd()
	importPath, _ := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), pwd)
	return filepath.ToSlash(importPath)
}
