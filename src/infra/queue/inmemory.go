package queue

import (
	"errors"
	"sync"

	"github.com/contre95/soulsolid/src/features/importing"
)

// InMemoryQueue is an in-memory implementation of the Queue interface
type InMemoryQueue struct {
	items sync.Map // map[string]importing.QueueItem
}

// NewInMemoryQueue creates a new in-memory queue
func NewInMemoryQueue() importing.Queue {
	return &InMemoryQueue{}
}

// Add adds a new item to the queue
func (q *InMemoryQueue) Add(item importing.QueueItem) error {
	if _, exists := q.items.Load(item.ID); exists {
		return importing.ErrAlreadyExists
	}
	q.items.Store(item.ID, item)
	return nil
}

// GetAll returns all items in the queue
func (q *InMemoryQueue) GetAll() map[string]importing.QueueItem {
	items := make(map[string]importing.QueueItem)
	q.items.Range(func(key, value any) bool {
		if item, ok := value.(importing.QueueItem); ok {
			if keyStr, ok := key.(string); ok {
				items[keyStr] = item
			}
		}
		return true
	})
	return items
}

// GetByID returns a specific item by ID
func (q *InMemoryQueue) GetByID(id string) (importing.QueueItem, error) {
	if value, ok := q.items.Load(id); ok {
		if item, ok := value.(importing.QueueItem); ok {
			return item, nil
		}
	}
	return importing.QueueItem{}, errors.New("item not found")
}

// Remove removes an item from the queue by ID
func (q *InMemoryQueue) Remove(id string) error {
	if _, ok := q.items.Load(id); !ok {
		return errors.New("item not found")
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
