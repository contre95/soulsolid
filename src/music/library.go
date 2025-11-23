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
