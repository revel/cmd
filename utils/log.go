package utils

import (
	"fmt"
	"os"
	"strings"

	"github.com/revel/cmd/logger"
	"github.com/revel/config"
)

var Logger = logger.New()

func InitLogger(basePath string, logLevel logger.LogLevel) {
	newContext := config.NewContext()
	if logLevel == logger.LvlDebug {
		newContext.SetOption("log.debug.output", "stdout")
		println("Debug on")
	} else {
		newContext.SetOption("log.debug.output", "off")
	}
	if logLevel >= logger.LvlInfo {
		newContext.SetOption("log.info.output", "stdout")
	} else {
		newContext.SetOption("log.inf.output", "off")
	}

	newContext.SetOption("log.warn.output", "stderr")
	newContext.SetOption("log.error.output", "stderr")
	newContext.SetOption("log.crit.output", "stderr")
	Logger.SetHandler(logger.InitializeFromConfig(basePath, newContext))
}

// This function is to throw a panic that may be caught by the packger so it can perform the needed
// imports.
func Retryf(format string, args ...interface{}) {
	// Ensure the user's command prompt starts on the next line.
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	panic(format) // Panic instead of os.Exit so that deferred will run.
}

type LoggedError struct{ error }

func NewLoggedError(err error) *LoggedError {
	return &LoggedError{err}
}
