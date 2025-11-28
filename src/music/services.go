package music

import "context"

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
}

// LibraryService defines the interface for library operations
type LibraryService interface {
	GetTracks(ctx context.Context) ([]*Track, error)
	GetTracksPaginated(ctx context.Context, limit, offset int) ([]*Track, error)
	GetTracksCount(ctx context.Context) (int, error)
	GetTrack(ctx context.Context, trackID string) (*Track, error)
	UpdateTrack(ctx context.Context, track *Track) error
}

// JobService defines the interface for job management
type JobService interface {
	StartJob(jobType string, name string, metadata map[string]any) (string, error)
	UpdateJobProgress(jobID string, progress int, message string)
}
