package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// The error is a wrapper for the.
type (
	SourceError struct {
		SourceType               string   // The type of source that failed to build.
		Title, Path, Description string   // Description of the error, as presented to the user.
		Line, Column             int      // Where the error was encountered.
		SourceLines              []string // The entire source file, split into lines.
		Stack                    string   // The raw stack trace string from debug.Stack().
		MetaError                string   // Error that occurred producing the error page.
		Link                     string   // A configurable link to wrap the error source in
	}
	SourceLine struct {
		Source  string
		Line    int
		IsError bool
	}
)

// Return a new error object.
func NewError(source, title, path, description string) *SourceError {
	return &SourceError{
		SourceType:  source,
		Title:       title,
		Path:        path,
		Description: description,
	}
}

// Creates a link based on the configuration setting "errors.link".
func (e *SourceError) SetLink(errorLink string) {
	errorLink = strings.ReplaceAll(errorLink, "{{Path}}", e.Path)
	errorLink = strings.ReplaceAll(errorLink, "{{Line}}", strconv.Itoa(e.Line))

	e.Link = "<a href=" + errorLink + ">" + e.Path + ":" + strconv.Itoa(e.Line) + "</a>"
}

// Error method constructs a plaintext version of the error, taking
// account that fields are optionally set. Returns e.g. Compilation Error
// (in views/header.html:51): expected right delim in end; got "}".
func (e *SourceError) Error() string {
	if e == nil {
		panic("opps")
	}
	loc := ""
	if e.Path != "" {
		line := ""
		if e.Line != 0 {
			line = fmt.Sprintf(":%d", e.Line)
		}
		loc = fmt.Sprintf("(in %s%s)", e.Path, line)
	}
	header := loc
	if e.Title != "" {
		if loc != "" {
			header = fmt.Sprintf("%s %s: ", e.Title, loc)
		} else {
			header = fmt.Sprintf("%s: ", e.Title)
		}
	}
	return fmt.Sprintf("%s%s", header, e.Description)
}

// ContextSource method returns a snippet of the source around
// where the error occurred.
func (e *SourceError) ContextSource() []SourceLine {
	if e.SourceLines == nil {
		return nil
	}
	start := (e.Line - 1) - 5
	if start < 0 {
		start = 0
	}
	end := (e.Line - 1) + 5
	if end > len(e.SourceLines) {
		end = len(e.SourceLines)
	}

	lines := make([]SourceLine, end-start)
	for i, src := range e.SourceLines[start:end] {
		fileLine := start + i + 1
		lines[i] = SourceLine{src, fileLine, fileLine == e.Line}
	}
	return lines
}
