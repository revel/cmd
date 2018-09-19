// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package harness

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
	"log"
)

// App contains the configuration for running a Revel app.  (Not for the app itself)
// Its only purpose is constructing the command to execute.
type App struct {
	BinaryPath string // Path to the app executable
	Port       int    // Port to pass as a command line argument.
	cmd        AppCmd // The last cmd returned.
	Paths      *model.RevelContainer
}

// NewApp returns app instance with binary path in it
func NewApp(binPath string, paths *model.RevelContainer) *App {
	return &App{BinaryPath: binPath, Paths: paths, Port: paths.HTTPPort}
}

// Cmd returns a command to run the app server using the current configuration.
func (a *App) Cmd(runMode string) AppCmd {
	a.cmd = NewAppCmd(a.BinaryPath, a.Port, runMode, a.Paths)
	return a.cmd
}

// Kill the last app command returned.
func (a *App) Kill() {
	a.cmd.Kill()
}

// AppCmd manages the running of a Revel app server.
// It requires revel.Init to have been called previously.
type AppCmd struct {
	*exec.Cmd
}

// NewAppCmd returns the AppCmd with parameters initialized for running app
func NewAppCmd(binPath string, port int, runMode string, paths *model.RevelContainer) AppCmd {
	cmd := exec.Command(binPath,
		fmt.Sprintf("-port=%d", port),
		fmt.Sprintf("-importPath=%s", paths.ImportPath),
		fmt.Sprintf("-runMode=%s", runMode))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return AppCmd{cmd}
}

// Start the app server, and wait until it is ready to serve requests.
func (cmd AppCmd) Start(c *model.CommandConfig) error {
	listeningWriter := &startupListeningWriter{os.Stdout, make(chan bool),c}
	cmd.Stdout = listeningWriter
	utils.Logger.Info("Exec app:", "path", cmd.Path, "args", cmd.Args)
	if err := cmd.Cmd.Start(); err != nil {
		utils.Logger.Fatal("Error running:", "error", err)
	}

	select {
	case exitState := <-cmd.waitChan():
		return errors.New("revel/harness: app died reason: " + exitState)

	case <-time.After(60 * time.Second):
		log.Println("Killing revel server process did not respond after wait timeout.", "processid", cmd.Process.Pid)
		cmd.Kill()
		return errors.New("revel/harness: app timed out")

	case <-listeningWriter.notifyReady:
		return nil
	}

	// TODO remove this unreachable code and document it
	panic("Impossible")
}

// Run the app server inline.  Never returns.
func (cmd AppCmd) Run() {
	log.Println("Exec app:", "path", cmd.Path, "args", cmd.Args)
	if err := cmd.Cmd.Run(); err != nil {
		utils.Logger.Fatal("Error running:", "error", err)
	}
}

// Kill terminates the app server if it's running.
func (cmd AppCmd) Kill() {
	if cmd.Cmd != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited()) {
		utils.Logger.Info("Killing revel server pid", "pid", cmd.Process.Pid)
		err := cmd.Process.Kill()
		if err != nil {
			utils.Logger.Fatal("Failed to kill revel server:", "error", err)
		}
	}
}

// Return a channel that is notified when Wait() returns.
func (cmd AppCmd) waitChan() <-chan string {
	ch := make(chan string, 1)
	go func() {
		_ = cmd.Wait()
		state := cmd.ProcessState
		exitStatus := " unknown "
		if state!=nil {
			exitStatus = state.String()
		}

		ch <- exitStatus
	}()
	return ch
}

// A io.Writer that copies to the destination, and listens for "Revel engine is listening on.."
// in the stream.  (Which tells us when the revel server has finished starting up)
// This is super ghetto, but by far the simplest thing that should work.
type startupListeningWriter struct {
	dest        io.Writer
	notifyReady chan bool
	c *model.CommandConfig
}

func (w *startupListeningWriter) Write(p []byte) (int, error) {
	if w.notifyReady != nil && bytes.Contains(p, []byte("Revel engine is listening on")) {
		w.notifyReady <- true
		w.notifyReady = nil
	}
	if w.c.HistoricMode {
		if w.notifyReady != nil && bytes.Contains(p, []byte("Listening on")) {
			w.notifyReady <- true
			w.notifyReady = nil
		}
	}
	return w.dest.Write(p)
}
