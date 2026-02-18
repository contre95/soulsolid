package importing

// QueueItemType represents the type of item in the queue
type QueueItemType string

const (
	ManualReview    QueueItemType = "manual_review"
	Duplicate       QueueItemType = "duplicate"
	FailedImport    QueueItemType = "failed_import"
	MissingMetadata QueueItemType = "missing_metadata"
)
