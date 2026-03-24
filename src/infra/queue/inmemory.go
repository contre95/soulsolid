package queue

import (
	"sync"

	"github.com/contre95/soulsolid/src/music"
)

// InMemoryQueue is an in-memory implementation of the Queue interface
type InMemoryQueue struct {
	items sync.Map // map[string]music.QueueItem
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue() music.Queue {
	return &InMemoryQueue{}
}

// Add adds a new item to the queue
func (q *InMemoryQueue) Add(item music.QueueItem) error {
	if _, exists := q.items.Load(item.ID); exists {
		return music.ErrTrackInTheQueueAlready
	}
	q.items.Store(item.ID, item)
	return nil
}

// GetAll returns all items in the queue
func (q *InMemoryQueue) GetAll() map[string]music.QueueItem {
	items := make(map[string]music.QueueItem)
	q.items.Range(func(key, value any) bool {
		if item, ok := value.(music.QueueItem); ok {
			if keyStr, ok := key.(string); ok {
				items[keyStr] = item
			}
		}
		return true
	})
	return items
}

// GetByID returns a specific item by ID
func (q *InMemoryQueue) GetByID(id string) (music.QueueItem, error) {
	if value, ok := q.items.Load(id); ok {
		if item, ok := value.(music.QueueItem); ok {
			return item, nil
		}
	}
	return music.QueueItem{}, music.ErrTrackNotFoundInQueue
}

// Remove removes an item from the queue by ID
func (q *InMemoryQueue) Remove(id string) error {
	if _, ok := q.items.Load(id); !ok {
		return music.ErrTrackNotFoundInQueue
	}
	q.items.Delete(id)
	return nil
}

// Clear removes all items from the queue
func (q *InMemoryQueue) Clear() error {
	q.items.Range(func(key, value any) bool {
		q.items.Delete(key)
		return true
	})
	return nil
}

// GetGroupedByArtist returns items grouped by primary artist
func (q *InMemoryQueue) GetGroupedByArtist() map[string][]music.QueueItem {
	allItems := q.GetAll()
	groups := make(map[string][]music.QueueItem)

	for _, item := range allItems {
		if item.Track != nil {
			if len(item.Track.Artists) > 0 && item.Track.Artists[0].Artist != nil {
				artistName := item.Track.Artists[0].Artist.Name
				groups[artistName] = append(groups[artistName], item)
			} else {
				// Fallback for tracks without artists
				unknownArtist := "Unknown Artist"
				groups[unknownArtist] = append(groups[unknownArtist], item)
			}
		}
		// Skip items without track
	}

	return groups
}

// GetGroupedByAlbum returns items grouped by album
func (q *InMemoryQueue) GetGroupedByAlbum() map[string][]music.QueueItem {
	allItems := q.GetAll()
	groups := make(map[string][]music.QueueItem)

	for _, item := range allItems {
		if item.Track != nil && item.Track.Album != nil {
			albumKey := item.Track.Album.Title
			if len(item.Track.Album.Artists) > 0 && item.Track.Album.Artists[0].Artist != nil {
				albumKey += " - " + item.Track.Album.Artists[0].Artist.Name
			}
			groups[albumKey] = append(groups[albumKey], item)
		} else {
			// Fallback for tracks without albums
			unknownAlbum := "Unknown Album"
			groups[unknownAlbum] = append(groups[unknownAlbum], item)
		}
	}

	return groups
}
