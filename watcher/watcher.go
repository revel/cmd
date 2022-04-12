// Copyright (c) 2012-2016 The Revel Framework Authors, All rights reserved.
// Revel Framework source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package watcher

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/revel/cmd/model"
	"github.com/revel/cmd/utils"
)

// Listener is an interface for receivers of filesystem events.
type Listener interface {
	// Refresh is invoked by the watcher on relevant filesystem events.
	// If the listener returns an error, it is served to the user on the current request.
	Refresh() *utils.SourceError
}

// DiscerningListener allows the receiver to selectively watch files.
type DiscerningListener interface {
	Listener
	WatchDir(info os.FileInfo) bool
	WatchFile(basename string) bool
}

// Watcher allows listeners to register to be notified of changes under a given
// directory.
type Watcher struct {
	// Parallel arrays of watcher/listener pairs.
	watchers            []*fsnotify.Watcher
	listeners           []Listener
	forceRefresh        bool
	eagerRefresh        bool
	serial              bool
	lastError           int
	notifyMutex         sync.Mutex
	paths               *model.RevelContainer
	refreshTimer        *time.Timer // The timer to countdown the next refresh
	timerMutex          *sync.Mutex // A mutex to prevent concurrent updates
	refreshChannel      chan *utils.SourceError
	refreshChannelCount int
	refreshInterval     time.Duration // The interval between refreshing builds
}

// Creates a new watched based on the container.
func NewWatcher(paths *model.RevelContainer, eagerRefresh bool) *Watcher {
	return &Watcher{
		forceRefresh:    true,
		lastError:       -1,
		paths:           paths,
		refreshInterval: time.Duration(paths.Config.IntDefault("watch.rebuild.delay", 1000)) * time.Millisecond,
		eagerRefresh: eagerRefresh ||
			paths.DevMode &&
				paths.Config.BoolDefault("watch", true) &&
				paths.Config.StringDefault("watch.mode", "normal") == "eager",
		timerMutex:          &sync.Mutex{},
		refreshChannel:      make(chan *utils.SourceError, 10),
		refreshChannelCount: 0,
	}
}

// Listen registers for events within the given root directories (recursively).
func (w *Watcher) Listen(listener Listener, roots ...string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		utils.Logger.Fatal("Watcher: Failed to create watcher", "error", err)
	}

	// Replace the unbuffered Event channel with a buffered one.
	// Otherwise multiple change events only come out one at a time, across
	// multiple page views.  (There appears no way to "pump" the events out of
	// the watcher)
	// This causes a notification when you do a check in go, since you are modifying a buffer in use
	watcher.Events = make(chan fsnotify.Event, 100)
	watcher.Errors = make(chan error, 10)

	// Walk through all files / directories under the root, adding each to watcher.
	for _, p := range roots {
		// is the directory / file a symlink?
		f, err := os.Lstat(p)
		if err == nil && f.Mode()&os.ModeSymlink == os.ModeSymlink {
			var realPath string
			realPath, err = filepath.EvalSymlinks(p)
			if err != nil {
				panic(err)
			}
			p = realPath
		}

		fi, err := os.Stat(p)
		if err != nil {
			utils.Logger.Fatal("Watcher: Failed to stat watched path", "path", p, "error", err)
			continue
		}

		// If it is a file, watch that specific file.
		if !fi.IsDir() {
			err = watcher.Add(p)
			if err != nil {
				utils.Logger.Fatal("Watcher: Failed to watch", "path", p, "error", err)
			}
			continue
		}

		watcherWalker := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				utils.Logger.Fatal("Watcher: Error walking path:", "error", err)
				return nil
			}

			if info.IsDir() {
				if dl, ok := listener.(DiscerningListener); ok {
					if !dl.WatchDir(info) {
						return filepath.SkipDir
					}
				}

				err = watcher.Add(path)
				if err != nil {
					utils.Logger.Fatal("Watcher: Failed to watch", "path", path, "error", err)
				}
			}
			return nil
		}

		// Else, walk the directory tree.
		err = utils.Walk(p, watcherWalker)
		if err != nil {
			utils.Logger.Fatal("Watcher: Failed to walk directory", "path", p, "error", err)
		}
	}

	if w.eagerRefresh {
		// Create goroutine to notify file changes in real time
		go w.NotifyWhenUpdated(listener, watcher)
	}

	w.watchers = append(w.watchers, watcher)
	w.listeners = append(w.listeners, listener)
}

// NotifyWhenUpdated notifies the watcher when a file event is received.
func (w *Watcher) NotifyWhenUpdated(listener Listener, watcher *fsnotify.Watcher) {
	for {
		select {
		case ev := <-watcher.Events:
			if w.rebuildRequired(ev, listener) {
				if w.serial {
					// Serialize listener.Refresh() calls.
					w.notifyMutex.Lock()

					if err := listener.Refresh(); err != nil {
						utils.Logger.Error("Watcher: Listener refresh reported error:", "error", err)
					}
					w.notifyMutex.Unlock()
				} else {
					// Run refresh in parallel
					go func() {
						if err := w.notifyInProcess(listener); err != nil {
							utils.Logger.Error("failed to notify",
								"error", err)
						}
					}()
				}
			}
		case <-watcher.Errors:
			continue
		}
	}
}

// Notify causes the watcher to forward any change events to listeners.
// It returns the first (if any) error returned.
func (w *Watcher) Notify() *utils.SourceError {
	if w.serial {
		// Serialize Notify() calls.
		w.notifyMutex.Lock()
		defer w.notifyMutex.Unlock()
	}

	for i, watcher := range w.watchers {
		listener := w.listeners[i]

		// Pull all pending events / errors from the watcher.
		refresh := false
		for {
			select {
			case ev := <-watcher.Events:
				if w.rebuildRequired(ev, listener) {
					refresh = true
				}
				continue
			case <-watcher.Errors:
				continue
			default:
				// No events left to pull
			}
			break
		}

		utils.Logger.Info("Watcher:Notify refresh state", "Current Index", i, " last error index", w.lastError,
			"force", w.forceRefresh, "refresh", refresh, "lastError", w.lastError == i)
		if w.forceRefresh || refresh || w.lastError == i {
			var err *utils.SourceError
			if w.serial {
				err = listener.Refresh()
			} else {
				err = w.notifyInProcess(listener)
			}
			if err != nil {
				w.lastError = i
				w.forceRefresh = true
				return err
			}

			w.lastError = -1
			w.forceRefresh = false
		}
	}

	return nil
}

// Build a queue for refresh notifications
// this will not return until one of the queue completes.
func (w *Watcher) notifyInProcess(listener Listener) (err *utils.SourceError) {
	shouldReturn := false
	// This code block ensures that either a timer is created
	// or that a process would be added the the h.refreshChannel
	func() {
		w.timerMutex.Lock()
		defer w.timerMutex.Unlock()
		// If we are in the process of a rebuild, forceRefresh will always be true
		w.forceRefresh = true
		if w.refreshTimer != nil {
			utils.Logger.Info("Found existing timer running, resetting")
			w.refreshTimer.Reset(w.refreshInterval)
			shouldReturn = true
			w.refreshChannelCount++
		} else {
			w.refreshTimer = time.NewTimer(w.refreshInterval)
		}
	}()

	// If another process is already waiting for the timer this one
	// only needs to return the output from the channel
	if shouldReturn {
		return <-w.refreshChannel
	}
	utils.Logger.Info("Waiting for refresh timer to expire")
	<-w.refreshTimer.C
	w.timerMutex.Lock()

	// Ensure the queue is properly dispatched even if a panic occurs
	defer func() {
		for x := 0; x < w.refreshChannelCount; x++ {
			w.refreshChannel <- err
		}
		w.refreshChannelCount = 0
		w.refreshTimer = nil
		w.timerMutex.Unlock()
	}()

	err = listener.Refresh()
	if err != nil {
		utils.Logger.Info("Watcher: Recording error last build, setting rebuild on", "error", err)
	} else {
		w.lastError = -1
		w.forceRefresh = false
	}
	utils.Logger.Info("Rebuilt, result", "error", err)
	return
}

func (w *Watcher) rebuildRequired(ev fsnotify.Event, listener Listener) bool {
	// Ignore changes to dotfiles.
	if strings.HasPrefix(filepath.Base(ev.Name), ".") {
		return false
	}

	if dl, ok := listener.(DiscerningListener); ok {
		if !dl.WatchFile(ev.Name) || ev.Op&fsnotify.Chmod == fsnotify.Chmod {
			return false
		}
	}
	return true
}
