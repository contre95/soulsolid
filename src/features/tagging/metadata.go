package tagging

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// SearchParams contains parameters for searching tracks in metadata providers
type SearchParams struct {
	TrackID     string
	AlbumArtist string
	Album       string
	Title       string
	Year        int
}

// MetadataProvider defines the interface for fetching metadata from external services
type MetadataProvider interface {
	// SearchTracks searches for tracks using metadata parameters and returns a list of matches
	SearchTracks(ctx context.Context, params SearchParams) ([]*music.Track, error)

	// Name returns the provider name
	Name() string

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}
