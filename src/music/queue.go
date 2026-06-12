package music

import (
	"errors"
	"slices"
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
	// MissingMetadata indicates a track that is missing a required metadata field
	// whose absence is not permitted by the import config
	MissingMetadata QueueItemType = "missing_metadata"
	// Duplicate indicates a track that is a duplicate of an existing track
	Duplicate QueueItemType = "duplicate"
	// FailedImport indicates a track that failed to import
	FailedImport QueueItemType = "failed_import"
	// ExistingLyrics indicates a track that already has lyrics
	ExistingLyrics QueueItemType = "existing_lyrics"
	// Lyric404 indicates a track where lyrics were not found (404)
	Lyric404 QueueItemType = "lyric_404"
	// FailedLyrics indicates a track where lyrics fetch failed due to error
	FailedLyrics QueueItemType = "failed_lyrics"
)

// QueueItem represents an item in the import queue. A single item can carry more than one
// type at once (e.g. a track that is both a Duplicate and MissingMetadata).
type QueueItem struct {
	ID        string            `json:"id"`
	Types     []QueueItemType   `json:"types"`
	Track     *Track            `json:"track"`
	Timestamp time.Time         `json:"timestamp"`
	JobID     string            `json:"job_id"`
	Metadata  map[string]string `json:"metadata"`
}

// HasType reports whether the item carries the given type.
func (q QueueItem) HasType(t QueueItemType) bool {
	return slices.Contains(q.Types, t)
}

// PrimaryType returns the item's first type, or "" when it has none. It is a convenience for
// callers (e.g. the lyrics queue) where an item only ever carries a single type.
func (q QueueItem) PrimaryType() QueueItemType {
	if len(q.Types) == 0 {
		return ""
	}
	return q.Types[0]
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
