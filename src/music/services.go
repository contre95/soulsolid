package music

import (
	"context"
	"log/slog"
	"time"
)

// LyricsSearchParams contains parameters for searching lyrics
type LyricsSearchParams struct {
	TrackID     string
	AlbumArtist string
	Album       string
	Title       string
	Artist      string
}

// LyricsProviderInfo contains information about a lyrics provider for the UI
type LyricsProviderInfo struct {
	Name        string
	DisplayName string
	Enabled     bool
}

// MetadataService defines the interface for tagging operations
type MetadataService interface {
	AddChromaprintAndAcoustID(ctx context.Context, trackID string) error
	GetTrackFileTags(ctx context.Context, trackID string) (*Track, error)
	UpdateTrackTags(ctx context.Context, trackID string, formData map[string]string) error
}

// LyricsService defines the interface for lyrics operations
type LyricsService interface {
	AddLyrics(ctx context.Context, trackID string, providerName string) error
	GetEnabledLyricsProviders() map[string]bool
	GetLyricsProvidersInfo() []LyricsProviderInfo
	SearchLyrics(ctx context.Context, trackID string, providerName string) (string, error)
	GetLyricsQueueItems() map[string]QueueItem
}

// JobService defines the interface for job management
type JobService interface {
	StartJob(jobType string, name string, metadata map[string]any) (string, error)
	UpdateJobProgress(jobID string, progress int, message string)
	GetJob(jobID string) (*Job, bool)
	CancelJob(jobID string) error
	GetJobs() []*Job
	ClearFinishedJobs() error
}

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a background job
type Job struct {
	ID         string
	Type       string
	Name       string
	Status     JobStatus
	Progress   int
	Message    string
	Error      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Metadata   map[string]any
	CancelFunc context.CancelFunc
	Logger     *slog.Logger
	LogPath    string
	Cancelled  bool // Track if job has been cancelled
}

// JobProgress represents a progress update for a job
type JobProgress struct {
	JobID    string
	Progress int
	Message  string
}
