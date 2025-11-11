package importing

import (
	"context"
	"time"
)

// Watcher defines the interface for file system watchers
type Watcher interface {
	Start(ctx context.Context, watchPath string) error
	Stop()
}

// FileEventType represents the type of file system event
type FileEventType string

const (
	FileCreated   FileEventType = "created"
	FileRemoved   FileEventType = "removed"
	FileModified  FileEventType = "modified"
)

// FileEvent represents a file system event
type FileEvent struct {
	Path      string
	EventType FileEventType
	Timestamp time.Time
}