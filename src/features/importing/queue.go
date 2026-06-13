package importing

import "github.com/contre95/soulsolid/src/music"

// QueueItemType represents the type of item in the queue

const (
	ManualReview    music.QueueItemType = "manual_review"
	MissingMetadata music.QueueItemType = "missing_metadata"
	Duplicate       music.QueueItemType = "duplicate"
	FailedImport    music.QueueItemType = "failed_import"
)
