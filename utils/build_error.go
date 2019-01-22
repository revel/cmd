package utils

import (
	"fmt"

	"github.com/revel/revel/logger"
)

type (
	BuildError struct {
		Stack   interface{}
		Message string
		Args    []interface{}
	}
)

// Returns a new builed error
func NewBuildError(message string, args ...interface{}) (b *BuildError) {
	Logger.Info(message, args...)
	b = &BuildError{}
	b.Message = message
	b.Args = args
	b.Stack = logger.NewCallStack()
	Logger.Info("Stack", "stack", b.Stack)
	return b
}

// Returns a new BuildError if err is not nil
func NewBuildIfError(err error, message string, args ...interface{}) (b error) {
	if err != nil {
		if berr, ok := err.(*BuildError); ok {
			// This is already a build error so just append the args
			berr.Args = append(berr.Args, args...)
			return berr
		} else {
			args = append(args, "error", err.Error())
			b = NewBuildError(message, args...)
		}
	}
	return
}

// BuildError implements Error() string
func (b *BuildError) Error() string {
	return fmt.Sprint(b.Message, b.Args)
}
