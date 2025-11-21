package metrics

import "context"

// LibraryMetrics provides analytics and reporting functionality for the music library.
type LibraryMetrics interface {
	// Genre analysis
	GetGenreDistribution(ctx context.Context) (map[string]int, error)

	// Metadata completeness analysis
	GetMetadataCompleteness(ctx context.Context) (MetadataCompletenessStats, error)

	// Audio format analysis
	GetFormatDistribution(ctx context.Context) (map[string]int, error)

	// Temporal analysis (tracks by year)
	GetYearDistribution(ctx context.Context) (map[string]int, error)

	// Lyrics presence analysis
	GetLyricsStats(ctx context.Context) (LyricsStats, error)

	// Specific metadata field counts
	GetTracksWithISRC(ctx context.Context) (int, error)
	GetTracksWithValidBPM(ctx context.Context) (int, error)
	GetTracksWithValidYear(ctx context.Context) (int, error)
	GetTracksWithValidGenre(ctx context.Context) (int, error)

	// Total counts
	GetTotalTracks(ctx context.Context) (int, error)
	GetTotalArtists(ctx context.Context) (int, error)
	GetTotalAlbums(ctx context.Context) (int, error)

	// Storage operations for cached metrics
	StoreMetric(ctx context.Context, metricType, key string, value int) error
	GetStoredMetrics(ctx context.Context, metricType string) ([]StoredMetric, error)
	ClearStoredMetrics(ctx context.Context) error
}

// MetadataCompletenessStats represents the completeness of metadata across tracks.
type MetadataCompletenessStats struct {
	Complete      int // Tracks with all required metadata
	MissingGenre  int // Tracks missing genre
	MissingYear   int // Tracks missing year
	MissingLyrics int // Tracks missing lyrics
}

// LyricsStats represents lyrics presence statistics.
type LyricsStats struct {
	WithLyrics    int // Tracks that have lyrics
	WithoutLyrics int // Tracks that don't have lyrics
}

// StoredMetric represents a cached metric stored in the database.
type StoredMetric struct {
	Type  string // The type of metric (e.g., "genre_counts", "lyrics_stats")
	Key   string // The metric key (e.g., genre name, "has_lyrics")
	Value int    // The metric value (count)
}
