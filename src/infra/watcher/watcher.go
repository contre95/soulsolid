package watcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/fsnotify/fsnotify"
)
const DEBOUNCE_SECS = 5

// Watcher monitors the download path for new files and emits events
type Watcher struct {
	watcher       *fsnotify.Watcher
	watchPath     string
	debounceTimer *time.Timer
	debounceMutex sync.Mutex
	running       bool
	stopChan      chan struct{}
	eventChan     chan importing.FileEvent
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

// Start begins watching the download path for file changes
func (w *Watcher) Start(ctx context.Context, watchPath string) error {
	w.watchPath = watchPath
	slog.Info("Starting file watcher", "path", watchPath)

	// Add the download path to watch
	if err := w.watcher.Add(watchPath); err != nil {
		return err
	}

	w.running = true

	// Start the event loop
	go w.watchLoop(ctx)

	slog.Info("File watcher started successfully")
	return nil
}

// Stop stops the file watcher
func (w *Watcher) Stop() {
	if !w.running {
		return
	}

	slog.Info("Stopping file watcher")
	w.running = false
	close(w.stopChan)

	// Cancel any pending debounce timer
	w.debounceMutex.Lock()
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
		w.debounceTimer = nil
	}
	w.debounceMutex.Unlock()

	w.watcher.Close()
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
	// Only process file creation events
	if event.Op&fsnotify.Create != fsnotify.Create {
		return
	}

	// Check if it's a supported audio file
	if !w.isSupportedFile(event.Name) {
		return
	}

	slog.Info("Detected new supported file", "file", event.Name)

	// Start or reset the debounce timer
	w.debounceMutex.Lock()
	defer w.debounceMutex.Unlock()

	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	w.debounceTimer = time.AfterFunc(time.Duration(DEBOUNCE_SECS)*time.Second, func() {
		w.emitDebounceEvent()
	})
}

// isSupportedFile checks if the file is a supported audio format
func (w *Watcher) isSupportedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExtensions := map[string]bool{
		".mp3":  true,
		".flac": true,
	}
	_, supported := supportedExtensions[ext]
	return supported
}

// emitDebounceEvent emits a file event after debounce period
func (w *Watcher) emitDebounceEvent() {
	event := importing.FileEvent{
		Path:      w.watchPath,
		EventType: importing.FileCreated,
		Timestamp: time.Now(),
	}

	select {
	case w.eventChan <- event:
		slog.Info("Emitted file event after debounce", "path", event.Path)
	default:
		slog.Warn("Event channel full, dropping file event", "path", event.Path)
	}
}
