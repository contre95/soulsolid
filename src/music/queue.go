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

const (
	// ManualReview indicates a track that needs manual review before import
	ManualReview QueueItemType = "manual_review"
	// Duplicate indicates a track that is a duplicate of an existing track
	Duplicate QueueItemType = "duplicate"
	// FailedImport indicates a track that failed to import
	FailedImport QueueItemType = "failed_import"
	// MissingMetadata indicates a track that is missing required metadata
	MissingMetadata QueueItemType = "missing_metadata"
	// ExistingLyrics indicates a track that already has lyrics
	ExistingLyrics QueueItemType = "existing_lyrics"
	// Lyric404 indicates a track where lyrics were not found (404)
	Lyric404 QueueItemType = "lyric_404"
	// FailedLyrics indicates a track where lyrics fetch failed due to error
	FailedLyrics QueueItemType = "failed_lyrics"
)

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
