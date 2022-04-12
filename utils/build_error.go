package utils

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/revel/cmd/logger"
)

type (
	BuildError struct {
		Stack   interface{}
		Message string
		Args    []interface{}
	}
)

// Returns a new builed error.
func NewBuildError(message string, args ...interface{}) (b *BuildError) {
	Logger.Info(message, args...)
	b = &BuildError{}
	b.Message = message
	b.Args = args
	b.Stack = logger.NewCallStack()
	Logger.Info("Stack", "stack", b.Stack)
	return b
}

// Returns a new BuildError if err is not nil.
func NewBuildIfError(err error, message string, args ...interface{}) (b error) {
	if err != nil {
		var berr *BuildError
		if errors.As(err, &berr) {
			// This is already a build error so just append the args
			berr.Args = append(berr.Args, args...)
			return berr
		}

		args = append(args, "error", err.Error())
		b = NewBuildError(message, args...)
	}

	return
}

// BuildError implements Error() string.
func (b *BuildError) Error() string {
	return fmt.Sprint(b.Message, b.Args)
}

// Parse the output of the "go build" command.
// Return a detailed Error.
func NewCompileError(importPath, errorLink string, err error) *SourceError {
	// Get the stack from the error

	errorMatch := regexp.MustCompile(`(?m)^([^:#]+):(\d+):(\d+:)? (.*)$`).
		FindSubmatch([]byte(err.Error()))
	if errorMatch == nil {
		errorMatch = regexp.MustCompile(`(?m)^(.*?):(\d+):\s(.*?)$`).FindSubmatch([]byte(err.Error()))

		if errorMatch == nil {
			Logger.Error("Failed to parse build errors", "error", err)
			return &SourceError{
				SourceType:  "Go code",
				Title:       "Go Compilation Error",
				Description: "See console for build error.",
			}
		}

		errorMatch = append(errorMatch, errorMatch[3])

		Logger.Error("Build errors", "errors", err)
	}

	// Read the source for the offending file.
	var (
		relFilename  = string(errorMatch[1]) // e.g. "src/revel/sample/app/controllers/app.go"
		absFilename  = relFilename
		line, _      = strconv.Atoi(string(errorMatch[2]))
		description  = string(errorMatch[4])
		compileError = &SourceError{
			SourceType:  "Go code",
			Title:       "Go Compilation Error",
			Path:        relFilename,
			Description: description,
			Line:        line,
		}
	)

	// errorLink := paths.Config.StringDefault("error.link", "")

	if errorLink != "" {
		compileError.SetLink(errorLink)
	}

	fileStr, err := ReadLines(absFilename)
	if err != nil {
		compileError.MetaError = absFilename + ": " + err.Error()
		Logger.Info("Unable to readlines "+compileError.MetaError, "error", err)
		return compileError
	}

	compileError.SourceLines = fileStr
	return compileError
}
