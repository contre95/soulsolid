package watcher

import (
	"time"
)

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