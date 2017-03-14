// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

// The command line tool for running Revel apps.
package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/agtorre/gocolorize"
    "github.com/revel/cmd/revel/util"
)


// Command structure cribbed from the genius organization of the "go" command.
type Command struct {
	Run                    func(args []string)
	UsageLine, Short, Long string
}

// Name returns command name from usage line
func (cmd *Command) Name() string {
	name := cmd.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

var commands = []*Command{
	cmdNew,
	cmdNewRaml,
	cmdRun,
	cmdBuild,
	cmdPackage,
	cmdClean,
	cmdTest,
	cmdVersion,
}

func main() {
	if runtime.GOOS == "windows" {
		gocolorize.SetPlain(true)
	}
	fmt.Fprintf(os.Stdout, gocolorize.NewColor("blue").Paint(header))
	flag.Usage = func() { usage(1) }
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || args[0] == "help" {
		if len(args) == 1 {
			usage(0)
		}
		if len(args) > 1 {
			for _, cmd := range commands {
				if cmd.Name() == args[1] {
					tmpl(os.Stdout, helpTemplate, cmd)
					return
				}
			}
		}
		usage(2)
	}

	// Commands use panic to abort execution when something goes wrong.
	// Panics are logged at the point of error.  Ignore those.
	defer func() {
		if err := recover(); err != nil {
			if _, ok := err.(util.LoggedError); !ok {
				// This panic was not expected / logged.
				panic(err)
			}
			os.Exit(1)
		}
	}()

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.Run(args[1:])
			return
		}
	}

	util.Errorf("unknown command %q\nRun 'revel help' for usage.\n", args[0])
}

const header = `~
~ revel! http://revel.github.io
~
`

const usageTemplate = `usage: revel command [arguments]

The commands are:
{{range .}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}

Use "revel help [command]" for more information.
`

var helpTemplate = `usage: revel {{.UsageLine}}
{{.Long}}
`

func usage(exitCode int) {
	tmpl(os.Stderr, usageTemplate, commands)
	os.Exit(exitCode)
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
