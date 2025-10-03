package importing

import (
	"errors"
	"time"

	"github.com/contre95/soulsolid/src/music"
)

// QueueItemType represents the type of item in the queue
type QueueItemType string

var (
	// My domain depends on this but I have not way of ensure the implementation behaves like it shuold.
	// Should I return (error, ErrAlreadyExists) ???
	ErrAlreadyExists = errors.New("Track already in the Queue.")
	ErrNotFount      = errors.New("Track was not found in Queue.")
)

const (
	ManualReview QueueItemType = "manual_review"
	Duplicate    QueueItemType = "duplicate"
)

// QueueItem represents an item in the import queue
type QueueItem struct {
	ID        string        `json:"id"`
	Type      QueueItemType `json:"type"`
	Track     *music.Track  `json:"track"`
	Timestamp time.Time     `json:"timestamp"`
	JobID     string        `json:"job_id"`
	// Store raw names instead of created objects to avoid premature creation
	ArtistNames []string `json:"artist_names"`
	AlbumTitle  string   `json:"album_title"`
	AlbumYear   int      `json:"album_year"`
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
}
