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
