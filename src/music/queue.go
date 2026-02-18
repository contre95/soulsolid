package music

import (
	"errors"
	"time"
)

var (
	ErrTrackInTheQueueAlready = errors.New("track already in queue")
	ErrTrackNotFoundInQueue   = errors.New("track not found in queue")
)

type QueueItemType string

// QueueItem represents an item in the import queue
type QueueItem struct {
	ID        string            `json:"id"`
	Type      QueueItemType     `json:"type"`
	Track     *Track            `json:"track"`
	Timestamp time.Time         `json:"timestamp"`
	JobID     string            `json:"job_id"`
	Metadata  map[string]string `json:"metadata"`
}

// Queue defines the interface for managing import queue items
type Queue interface {
	// Add adds a new item to the queue, return ErrAlreadyExists if already in Queue.
	Add(item QueueItem) error
	// GetAll returns all items in the queue
	GetAll() map[string]QueueItem
	// GetByID returns a specific item by ID
	GetByID(id string) (QueueItem, error)
	// Remove removes an item from the queue by ID
	Remove(id string) error
	// Clear removes all items from the queue
	Clear() error
	// GetGroupedByArtist returns items grouped by primary artist
	GetGroupedByArtist() map[string][]QueueItem
	// GetGroupedByAlbum returns items grouped by album
	GetGroupedByAlbum() map[string][]QueueItem
}
