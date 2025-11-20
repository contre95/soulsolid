package music

import (
	"context"
)

// Library is the interface for managing the music library.
// It's our primary repository interface for the library domain.
type Library interface {
	// Track methods
	AddTrack(ctx context.Context, track *Track) error
	GetTrack(ctx context.Context, id string) (*Track, error)
	UpdateTrack(ctx context.Context, track *Track) error
	GetTracks(ctx context.Context) ([]*Track, error)
	GetTracksPaginated(ctx context.Context, limit, offset int) ([]*Track, error)
	GetTracksCount(ctx context.Context) (int, error)
	FindTrackByMetadata(ctx context.Context, title, artistName, albumTitle string) (*Track, error)
	FindTrackByPath(ctx context.Context, path string) (*Track, error)
	UpdateTrackFingerprint(ctx context.Context, trackID, fingerprint string) error

	// Album methods
	AddAlbum(ctx context.Context, album *Album) error
	UpdateAlbum(ctx context.Context, album *Album) error
	GetAlbum(ctx context.Context, id string) (*Album, error)
	GetAlbums(ctx context.Context) ([]*Album, error)
	GetAlbumsPaginated(ctx context.Context, limit, offset int) ([]*Album, error)
	GetAlbumsCount(ctx context.Context) (int, error)
	GetAlbumByArtistAndName(ctx context.Context, artistID, name string) (*Album, error)
	FindOrCreateAlbum(ctx context.Context, artist *Artist, albumTitle string, year int) (*Album, error)

	// Artist methods
	AddArtist(ctx context.Context, artist *Artist) error
	GetArtist(ctx context.Context, id string) (*Artist, error)
	GetArtists(ctx context.Context) ([]*Artist, error)
	GetArtistsPaginated(ctx context.Context, limit, offset int) ([]*Artist, error)
	GetArtistsCount(ctx context.Context) (int, error)
	GetArtistByName(ctx context.Context, name string) (*Artist, error)
	FindOrCreateArtist(ctx context.Context, artistName string) (*Artist, error)
}

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
