// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package util

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/revel/revel"
)
const (
	// RevelCmdImportPath Revel framework cmd tool import path
	RevelCmdImportPath = "github.com/revel/cmd"

	// DefaultRunMode for revel's application
	DefaultRunMode = "dev"
)

// LoggedError is wrapper to differentiate logged panics from unexpected ones.
type LoggedError struct{ error }

func Errorf(format string, args ...interface{}) {
	// Ensure the user's command prompt starts on the next line.
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	panic(LoggedError{}) // Panic instead of os.Exit so that deferred will run.
}

func PanicOnError(err error, msg string) {
	if revErr, ok := err.(*revel.Error); (ok && revErr != nil) || (!ok && err != nil) {
		fmt.Fprintf(os.Stderr, "Abort: %s: %s\n", msg, err)
		panic(LoggedError{err})
	}
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

func MustRenderTemplate(destPath, srcPath string, data map[string]interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	PanicOnError(err, "Failed to parse template "+srcPath)

	f, err := os.Create(destPath)
	PanicOnError(err, "Failed to create "+destPath)

	err = tmpl.Execute(f, data)
	PanicOnError(err, "Failed to render template "+srcPath)

	err = f.Close()
	PanicOnError(err, "Failed to close "+f.Name())
}

func MustChmod(filename string, mode os.FileMode) {
	err := os.Chmod(filename, mode)
	PanicOnError(err, fmt.Sprintf("Failed to chmod %d %q", mode, filename))
}

// copyDir copies a directory tree over to a new directory.  Any files ending in
// ".template" are treated as a Go template and rendered using the given data.
// Additionally, the trailing ".template" is stripped from the file name.
// Also, dot files and dot directories are skipped.
func MustCopyDir(destDir, srcDir string, data map[string]interface{}) error {
	return revel.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
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

func MustTarGzDir(destFilename, srcDir string,addFile func(fileToAdd string) bool) string {
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

	_ = revel.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if info.IsDir() || (addFile!=nil && !addFile(srcPath)) {
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
		Errorf("error opening directory: %s", err)
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
