package watcher

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/fsnotify/fsnotify"
)

const debounceSeconds = 5

// Watcher monitors the download path for new files and emits events
type Watcher struct {
	watcher       *fsnotify.Watcher
	watchPath     string
	debounceTimer *time.Timer
	debounceMutex sync.Mutex
	running       atomic.Bool
	stopChan      chan struct{}
	eventChan     chan importing.FileEvent
	lastFile      string
}

// NewWatcher creates a new file system watcher
func NewWatcher() (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:   watcher,
		eventChan: make(chan importing.FileEvent, 10),
		stopChan:  make(chan struct{}),
	}, nil
}

// GetEventChan returns a receive-only channel for reading file events
func (w *Watcher) GetEventChan() <-chan importing.FileEvent {
	return w.eventChan
}

// IsRunning returns whether the watcher is currently running
func (w *Watcher) IsRunning() bool {
	return w.running.Load()
}

// Start begins watching the download path for file changes
func (w *Watcher) Start(ctx context.Context, watchPath string) error {
	w.watchPath = watchPath
	slog.Info("Starting file watcher", "path", watchPath)

	// Recreate watcher if closed
	if w.watcher == nil {
		slog.Debug("Recreating fsnotify watcher")
		var err error
		w.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return err
		}
	}

	// Recreate eventChan if closed
	if w.eventChan == nil {
		slog.Debug("Recreating event channel")
		w.eventChan = make(chan importing.FileEvent, 10)
	}

	// Recreate stopChan
	slog.Debug("Recreating stop channel")
	w.stopChan = make(chan struct{})

	// Add the download path to watch
	slog.Debug("Adding root watch path", "path", watchPath)
	if err := w.watcher.Add(watchPath); err != nil {
		return err
	}

	// Add all subdirectories recursively
	slog.Debug("Walking subdirectories for recursive watching", "root", watchPath)
	filepath.WalkDir(watchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("Error walking directory", "path", path, "error", err)
			return err
		}
		if d.IsDir() && path != watchPath {
			slog.Debug("Adding subdirectory to watcher", "path", path)
			w.watcher.Add(path)
		}
		return nil
	})

	w.running.Store(true)

	// Start the event loop
	go w.watchLoop(ctx)

	slog.Info("File watcher started successfully")
	return nil
}

// Stop stops the file watcher
func (w *Watcher) Stop() {
	if !w.running.Load() {
		slog.Debug("Watcher already stopped")
		return
	}

	slog.Info("Stopping file watcher")
	w.running.Store(false)
	slog.Debug("Closing stop channel")
	close(w.stopChan)

	// Cancel any pending debounce timer
	w.debounceMutex.Lock()
	if w.debounceTimer != nil {
		slog.Debug("Stopping debounce timer")
		w.debounceTimer.Stop()
		w.debounceTimer = nil
	}
	w.debounceMutex.Unlock()

	slog.Debug("Closing fsnotify watcher")
	w.watcher.Close()
	w.watcher = nil
	slog.Debug("Closing event channel")
	close(w.eventChan)
	w.eventChan = nil
}

// watchLoop processes file system events
func (w *Watcher) watchLoop(ctx context.Context) {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "error", err)

		case <-w.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// handleEvent processes a single file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	slog.Debug("Handling fsnotify event", "op", event.Op, "name", event.Name)
	// Only process file creation events
	if event.Op&fsnotify.Create == fsnotify.Create {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			// Add all subdirectories recursively
			filepath.WalkDir(event.Name, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					slog.Error("Error walking directory", "path", path, "error", err)
					return err
				}
				if d.IsDir() && path != event.Name {
					slog.Debug("Adding subdirectory to watcher", "path", path)
					w.watcher.Add(path)
				}
				return nil
			})
			slog.Debug("Detected new directory, adding to watcher", "dir", event.Name)
			w.watcher.Add(event.Name)
		} else {
			slog.Info("Detected new file", "file", event.Name)

			// Start or reset the debounce timer
			w.debounceMutex.Lock()
			w.lastFile = event.Name
			defer w.debounceMutex.Unlock()

			if w.debounceTimer != nil {
				w.debounceTimer.Stop()
			}

			w.debounceTimer = time.AfterFunc(time.Duration(debounceSeconds)*time.Second, func() {
				w.emitDebounceEvent()
			})
		}
	} else {
		slog.Debug("Ignoring non-create event", "op", event.Op, "name", event.Name)
	}
}



// emitDebounceEvent emits a file event after debounce period
func (w *Watcher) emitDebounceEvent() {
	slog.Debug("Emitting debounced file event", "file", w.lastFile)
	if !w.running.Load() {
		slog.Debug("Watcher not running, skipping emit")
		return
	}
	event := importing.FileEvent{
		Path:      w.lastFile,
		EventType: importing.FileCreated,
		Timestamp: time.Now(),
	}

	select {
	case w.eventChan <- event:
		slog.Info("Emitted file event after debounce", "path", event.Path)
	default:
		slog.Debug("Event channel full, dropping file event", "path", event.Path)
	}
}
