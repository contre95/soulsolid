package music

import (
	"context"
)

// FileManager handles file operations for music tracks.
type FileManager interface {
	// GetLibraryPath generates the library path for a track without moving it.
	GetLibraryPath(ctx context.Context, track *Track) (string, error)
	// MoveTrackToLibrary moves a track to a new location based on its metadata.
	MoveTrackToLibrary(ctx context.Context, track *Track) (string, error)
	// MoveTrackFile moves a track file to an explicit destination path.
	MoveTrackFile(ctx context.Context, srcPath, destPath string) (string, error)
	// CopyTrackToLibrary copies a track to a new location based on its metadata.
	CopyTrackToLibrary(ctx context.Context, track *Track) (string, error)
	// DeleteTrack removes a track file from the library
	DeleteTrack(ctx context.Context, trackPath string) error
}

// PathOptions contains configuration options for file organization.
type PathOptions struct {
	Compilations    string
	AlbumSoundtrack string
	AlbumSingle     string
	AlbumEP         string
	DefaultPath     string
}
