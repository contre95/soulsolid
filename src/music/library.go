package music

import (
	"context"
)

// TrackFilter represents the filter criteria for tracks.
type TrackFilter struct {
	Title       string
	ArtistIDs   []string
	AlbumIDs    []string
	TextSearch  string // OR-match across track title, artist name, and album title
	Genre       string // exact genre match
	HasAcoustID *bool  // nil=any, true=has acoustid, false=missing
	LyricsFilter string // "": any, "has": has_lyrics=true AND lyrics not empty, "empty": has_lyrics=true AND lyrics empty, "instrumental": has_lyrics=false
	LyricsText  string // LIKE search within lyrics content
	AddedAfter  string // "": any, else "YYYY-MM-DD"; matches tracks added on or after this date (inclusive)
	AddedBefore string // "": any, else "YYYY-MM-DD"; matches tracks added on or before this date (inclusive)
}

// AlbumSort is the ordering used when browsing albums.
type AlbumSort string

const (
	AlbumSortArtist AlbumSort = "artist" // album artist, then year, then title
	AlbumSortTitle  AlbumSort = "title"
	AlbumSortYear   AlbumSort = "year"   // newest first
	AlbumSortRecent AlbumSort = "recent" // most recently added tracks first
)

// AlbumFilter represents the filter criteria for browsing albums.
type AlbumFilter struct {
	TextSearch string    // OR-match across album title and album artist name
	Genre      string    // albums containing at least one track with exactly this genre
	Sort       AlbumSort // defaults to AlbumSortArtist
}

// AlbumSummary is a lightweight, read-only view of an album used to browse the library.
// It is computed in a single query, so it avoids loading every track of every album.
type AlbumSummary struct {
	ID           string
	Title        string
	Artists      string // album artist names, comma separated
	Year         int    // release year, falling back to the newest track year; 0 when unknown
	TrackCount   int
	Duration     int    // total duration in seconds
	CoverTrackID string // first track of the album, used to serve the artwork; empty for albums without tracks
}

// Library is the interface for managing the music library.
// It's our primary repository interface for the library domain.
type Library interface {
	// Track methods
	AddTrack(ctx context.Context, track *Track) error
	GetTrack(ctx context.Context, id string) (*Track, error)
	UpdateTrack(ctx context.Context, track *Track) error
	DeleteTrack(ctx context.Context, id string) error
	GetTracks(ctx context.Context) ([]*Track, error)
	GetTracksPaginated(ctx context.Context, limit, offset int) ([]*Track, error)
	GetTracksFilteredPaginated(ctx context.Context, limit, offset int, filter *TrackFilter) ([]*Track, error)
	GetTracksCount(ctx context.Context) (int, error)
	GetTracksFilteredCount(ctx context.Context, filter *TrackFilter) (int, error)
	FindTrackByMetadata(ctx context.Context, title, artistName, albumTitle string) (*Track, error)
	FindTrackByPath(ctx context.Context, path string) (*Track, error)
	GetKeptDuplicates(ctx context.Context, trackID string) ([]*Track, error)

	// Album methods
	AddAlbum(ctx context.Context, album *Album) error
	UpdateAlbum(ctx context.Context, album *Album) error
	DeleteAlbum(ctx context.Context, id string) error
	GetAlbum(ctx context.Context, id string) (*Album, error)
	GetAlbums(ctx context.Context) ([]*Album, error)
	GetAlbumsPaginated(ctx context.Context, limit, offset int) ([]*Album, error)
	GetAlbumsFilteredPaginated(ctx context.Context, limit, offset int, titleFilter string, artistIDs []string) ([]*Album, error)
	GetAlbumsCount(ctx context.Context) (int, error)
	GetAlbumsFilteredCount(ctx context.Context, titleFilter string, artistIDs []string) (int, error)
	SearchAlbums(ctx context.Context, query string, limit, offset int) ([]*Album, error)
	GetAlbumSummaries(ctx context.Context, limit, offset int, filter *AlbumFilter) ([]*AlbumSummary, error)
	GetAlbumSummariesCount(ctx context.Context, filter *AlbumFilter) (int, error)
	GetAlbumTracks(ctx context.Context, albumID string) ([]*Track, error) // ordered by disc and track number
	GetGenres(ctx context.Context) ([]string, error)
	GetAlbumByArtistAndName(ctx context.Context, artistID, name string) (*Album, error)
	FindOrCreateAlbum(ctx context.Context, artist *Artist, albumTitle string, year int) (*Album, error)

	// Artist methods
	AddArtist(ctx context.Context, artist *Artist) error
	DeleteArtist(ctx context.Context, id string) error
	GetArtist(ctx context.Context, id string) (*Artist, error)
	GetArtists(ctx context.Context) ([]*Artist, error)
	GetArtistsPaginated(ctx context.Context, limit, offset int) ([]*Artist, error)
	GetArtistsFilteredPaginated(ctx context.Context, limit, offset int, nameFilter string) ([]*Artist, error)
	GetArtistsCount(ctx context.Context) (int, error)
	GetArtistsFilteredCount(ctx context.Context, nameFilter string) (int, error)
	GetArtistByName(ctx context.Context, name string) (*Artist, error)
	FindOrCreateArtist(ctx context.Context, artistName string) (*Artist, error)
}
