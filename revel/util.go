package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/revel/revel"
)

// Use a wrapper to differentiate logged panics from unexpected ones.
type LoggedError struct{ error }

func panicOnError(err error, msg string) {
	if revErr, ok := err.(*revel.Error); (ok && revErr != nil) || (!ok && err != nil) {
		fmt.Fprintf(os.Stderr, "Abort: %s: %s\n", msg, err)
		panic(LoggedError{err})
	}
}

func mustCopyFile(destFilename, srcFilename string) {
	destFile, err := os.Create(destFilename)
	panicOnError(err, "Failed to create file "+destFilename)

	srcFile, err := os.Open(srcFilename)
	panicOnError(err, "Failed to open file "+srcFilename)

	_, err = io.Copy(destFile, srcFile)
	panicOnError(err,
		fmt.Sprintf("Failed to copy data from %s to %s", srcFile.Name(), destFile.Name()))

	err = destFile.Close()
	panicOnError(err, "Failed to close file "+destFile.Name())

	err = srcFile.Close()
	panicOnError(err, "Failed to close file "+srcFile.Name())
}

func mustRenderTemplate(destPath, srcPath string, data map[string]interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	panicOnError(err, "Failed to parse template "+srcPath)

	f, err := os.Create(destPath)
	panicOnError(err, "Failed to create "+destPath)

	err = tmpl.Execute(f, data)
	panicOnError(err, "Failed to render template "+srcPath)

	err = f.Close()
	panicOnError(err, "Failed to close "+f.Name())
}

func mustChmod(filename string, mode os.FileMode) {
	err := os.Chmod(filename, mode)
	panicOnError(err, fmt.Sprintf("Failed to chmod %d %q", mode, filename))
}

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".template" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".template" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func mustCopyDir(destDir, srcDir string, data map[string]interface{}) error {
	var fullSrcDir string
	// Handle symlinked directories.
	f, err := os.Lstat(srcDir)
	if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
		fullSrcDir, err = filepath.EvalSymlinks(srcDir)
		if err != nil {
			panic(err)
		}
	} else {
		fullSrcDir = srcDir
	}

	return filepath.Walk(fullSrcDir, func(srcPath string, info os.FileInfo, err error) error {
		// Get the relative path from the source base, and the corresponding path in
		// the dest directory.
		relSrcPath := strings.TrimLeft(srcPath[len(fullSrcDir):], string(os.PathSeparator))
		destPath := path.Join(destDir, relSrcPath)

		// Skip dot files and dot directories.
		if strings.HasPrefix(relSrcPath, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// if this is a link, call mustCopyDir recursively on the actual directory
		link, err := os.Lstat(srcPath)
		if err == nil && link.Mode()&os.ModeSymlink == os.ModeSymlink {
			// lookup the actual directory
			realSrcPath, err := filepath.EvalSymlinks(srcPath)
			panicOnError(err, "Failed to read symlink")

			real_info, err := os.Stat(realSrcPath)
			panicOnError(err, "Failed to stat symlink target")

			// TODO should factor out this and the same code below for regular files

			if real_info.IsDir() {
				// make the destination sub-directory
				err = os.MkdirAll(path.Join(destPath), 0777)
				if !os.IsExist(err) {
					panicOnError(err, "Failed to create directory")
				}

				// copy the actual directory
				// TODO, this is the exception to the factoring out code idea
				// we need to manually call mustCopyDir on a symlink directory
				// because filepath.Walk does not traverse symlink directories
				mustCopyDir(destPath, realSrcPath, data)
				return nil
			}

			// If this file ends in ".template", render it as a template.
			if strings.HasSuffix(realSrcPath, ".template") {
				mustRenderTemplate(destPath[:len(destPath)-len(".template")], realSrcPath, data)
				return nil
			}

			// Else, just copy it over.
			mustCopyFile(destPath, realSrcPath)

		}

		// TODO should factor out this and the same code above for symlink files
		// Create a subdirectory if necessary.
		if info.IsDir() {
			err := os.MkdirAll(path.Join(destDir, relSrcPath), 0777)
			if !os.IsExist(err) {
				panicOnError(err, "Failed to create directory")
			}
			return nil
		}

		// If this file ends in ".template", render it as a template.
		if strings.HasSuffix(relSrcPath, ".template") {
			mustRenderTemplate(destPath[:len(destPath)-len(".template")], srcPath, data)
			return nil
		}

		// Else, just copy it over.
		mustCopyFile(destPath, srcPath)
		return nil
	})
}

func mustTarGzDir(destFilename, srcDir string) string {
	zipFile, err := os.Create(destFilename)
	panicOnError(err, "Failed to create archive")
	defer zipFile.Close()

	gzipWriter := gzip.NewWriter(zipFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		srcFile, err := os.Open(srcPath)
		panicOnError(err, "Failed to read source file")
		defer srcFile.Close()

		err = tarWriter.WriteHeader(&tar.Header{
			Name:    strings.TrimLeft(srcPath[len(srcDir):], string(os.PathSeparator)),
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		})
		panicOnError(err, "Failed to write tar entry header")

		_, err = io.Copy(tarWriter, srcFile)
		panicOnError(err, "Failed to copy")

		return nil
	})

	return zipFile.Name()
}

func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// empty returns true if the given directory is empty.
// the directory must exist.
func empty(dirname string) bool {
	dir, err := os.Open(dirname)
	if err != nil {
		errorf("error opening directory: %s", err)
	}
	defer dir.Close()
	results, _ := dir.Readdir(1)
	return len(results) == 0
}
