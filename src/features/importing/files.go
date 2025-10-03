package importing

import (
	"context"

	"soulsolid/src/music"
)

type FileOrganizer interface {
	// GetLibraryPath generates the library path for a track without moving it.
	GetLibraryPath(ctx context.Context, track *music.Track) (string, error)
	// MoveTrack moves a track to a new location based on its metadata.
	MoveTrack(ctx context.Context, track *music.Track) (string, error)
	// CopyTrack copies a track to a new location based on its metadata.
	CopyTrack(ctx context.Context, track *music.Track) (string, error)
	// DeleteTrack removes a track file from the library
	DeleteTrack(ctx context.Context, trackPath string) error
}

// PathOptions contains configuration options for file organization.
type Paths struct {
	Compilations    string
	AlbumSoundtrack string
	AlbumSingle     string
	AlbumEP         string
	DefaultPath     string
}
