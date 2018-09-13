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

	"github.com/jessevdk/go-flags"

	"github.com/agtorre/gocolorize"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"github.com/revel/cmd/logger"
	"os/exec"
	"path/filepath"
	"go/build"
)

const (
	// RevelCmdImportPath Revel framework cmd tool import path
	RevelCmdImportPath = "github.com/revel/cmd"

	// DefaultRunMode for revel's application
	DefaultRunMode = "dev"
)

// Command structure cribbed from the genius organization of the "go" command.
type Command struct {
	UpdateConfig               func(c *model.CommandConfig, args []string) bool
	RunWith                    func(c *model.CommandConfig)
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

// The constants
const (
	NEW model.COMMAND = iota +1
	RUN
	BUILD
	PACAKAGE
	CLEAN
	TEST
	VERSION
)
// The commands
var commands = []*Command{
	nil, // Safety net, prevent missing index from running
	cmdNew,
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
	c := &model.CommandConfig{}
	wd,_ := os.Getwd()

	utils.InitLogger(wd,logger.LvlError)

	parser := flags.NewParser(c, flags.HelpFlag | flags.PassDoubleDash)
	if ini:=flag.String("ini","none","");*ini!="none" {
		if err:=flags.NewIniParser(parser).ParseFile(*ini);err!=nil {
			utils.Logger.Error("Unable to load ini", "error",err)
		}
	} else {
		if _, err := parser.Parse(); err != nil {
			utils.Logger.Info("Command line options failed", "error", err.Error())

			// Decode nature of error
			if perr,ok:=err.(*flags.Error); ok {
				if perr.Type == flags.ErrRequired {
					// Try the old way
					if !main_parse_old(c) {
						println("Command line error:", err.Error())
						parser.WriteHelp(os.Stdout)
						os.Exit(1)
					}
				} else {
					println("Command line error:", err.Error())
					parser.WriteHelp(os.Stdout)
					os.Exit(1)
				}
			} else {
				println("Command line error:", err.Error())
				parser.WriteHelp(os.Stdout)
				os.Exit(1)
			}
		} else {
			switch parser.Active.Name {
			case "new":
				c.Index = NEW
			case "run":
				c.Index = RUN
			case "build":
				c.Index = BUILD
			case "package":
				c.Index = PACAKAGE
			case "clean":
				c.Index = CLEAN
			case "test":
				c.Index = TEST
			case "version":
				c.Index = VERSION
			}
		}
	}

	// Switch based on the verbose flag
	if c.Verbose {
		utils.InitLogger(wd, logger.LvlDebug)
	} else {
		utils.InitLogger(wd, logger.LvlWarn)
	}
	println("Revel executing:", commands[c.Index].Short)
	// checking and setting go paths
	initGoPaths(c)

	commands[c.Index].RunWith(c)


}

// Try to populate the CommandConfig using the old techniques
func main_parse_old(c *model.CommandConfig) bool {
	// Take the old command format and try to parse them
	flag.Usage = func() { usage(1) }
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 || args[0] == "help" {
		if len(args) == 1 {
			usage(0)
		}

		if len(args) > 1 {
			for _, cmd := range commands {
				if cmd!=nil && cmd.Name() == args[1] {
					tmpl(os.Stdout, helpTemplate, cmd)
					return false
				}
			}
		}
		usage(2)
	}

	for _, cmd := range commands {
		if cmd!=nil && cmd.Name() == args[0] {
			println("Running", cmd.Name())
			return cmd.UpdateConfig(c, args[1:])
		}
	}

	return false
}

func main_old() {
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
			if _, ok := err.(utils.LoggedError); !ok {
				// This panic was not expected / logged.
				panic(err)
			}
			os.Exit(1)
		}
	}()

	//for _, cmd := range commands {
	//	if cmd.Name() == args[0] {
	//		cmd.UpdateConfig(args[1:])
	//		return
	//	}
	//}

	utils.Logger.Fatalf("unknown command %q\nRun 'revel help' for usage.\n", args[0])
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

// lookup and set Go related variables
func initGoPaths(c *model.CommandConfig) {
	// lookup go path
	c.GoPath = build.Default.GOPATH
	if c.GoPath == "" {
		utils.Logger.Fatal("Abort: GOPATH environment variable is not set. " +
			"Please refer to http://golang.org/doc/code.html to configure your Go environment.")
	}

	// check for go executable
	var err error
	c.GoCmd, err = exec.LookPath("go")
	if err != nil {
		utils.Logger.Fatal("Go executable not found in PATH.")
	}

	// revel/revel#1004 choose go path relative to current working directory
	workingDir, _ := os.Getwd()
	goPathList := filepath.SplitList(c.GoPath)
	for _, path := range goPathList {
		if strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			c.SrcRoot = path
			break
		}

		path, _ = filepath.EvalSymlinks(path)
		if len(path) > 0 && strings.HasPrefix(strings.ToLower(workingDir), strings.ToLower(path)) {
			c.SrcRoot = path
			break
		}
	}

	if len(c.SrcRoot) == 0 {
		utils.Logger.Fatal("Abort: could not create a Revel application outside of GOPATH.")
	}

	// set go src path
	c.SrcRoot = filepath.Join(c.SrcRoot, "src")
}