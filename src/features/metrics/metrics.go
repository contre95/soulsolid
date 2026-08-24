package metrics

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// LibraryMetrics provides analytics and reporting functionality for the music library.
type LibraryMetrics interface {
	// Genre analysis
	GetGenreDistribution(ctx context.Context) (map[string]int, error)

	// Metadata completeness analysis
	GetMetadataCompleteness(ctx context.Context) (music.MetadataCompletenessStats, error)

	// Audio format analysis
	GetFormatDistribution(ctx context.Context) (map[string]int, error)

	// Temporal analysis (tracks by year)
	GetYearDistribution(ctx context.Context) (map[string]int, error)

	// Lyrics presence analysis
	GetLyricsStats(ctx context.Context) (music.LyricsStats, error)

	// Specific metadata field counts
	GetTracksWithISRC(ctx context.Context) (int, error)
	GetTracksWithValidBPM(ctx context.Context) (int, error)
	GetTracksWithValidYear(ctx context.Context) (int, error)
	GetTracksWithValidGenre(ctx context.Context) (int, error)
	GetTracksWithAcoustID(ctx context.Context) (int, error)
	GetTracksWithChromaprint(ctx context.Context) (int, error)

	// Total counts
	GetTotalTracks(ctx context.Context) (int, error)
	GetTotalArtists(ctx context.Context) (int, error)
	GetTotalAlbums(ctx context.Context) (int, error)

	// Storage operations for cached metrics
	StoreMetric(ctx context.Context, metricType, key string, value int) error
	GetStoredMetrics(ctx context.Context, metricType string) ([]music.StoredMetric, error)
	ClearStoredMetrics(ctx context.Context) error
}
