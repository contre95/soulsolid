package streaming

import "context"

// QueueLocator resolves a pending (not yet imported) track file path from the import queue.
type QueueLocator interface {
	GetPendingTrackPath(itemID string) (string, error)
}

// LibraryLocator resolves an imported track file path from the library.
type LibraryLocator interface {
	GetLibraryTrackPath(ctx context.Context, trackID string) (string, error)
}
