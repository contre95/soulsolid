package metadata

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
	AcoustID    string
}

// AlbumSearchParams contains parameters for looking up an album's release info.
type AlbumSearchParams struct {
	Album          string
	AlbumArtist    string
	Barcode        string
	ReleaseGroupID string // MusicBrainz release group ID, when known
}

// AlbumInfo is the official release info of an album as reported by a provider.
type AlbumInfo struct {
	Title       string
	Artist      string
	TotalTracks int // across all discs
	Source      string
	SourceURL   string
}

// AlbumMetadataProvider is an optional capability of a MetadataProvider: looking up the
// official track count of an album. FetchAlbumInfo returns (nil, nil) when nothing matches.
type AlbumMetadataProvider interface {
	FetchAlbumInfo(ctx context.Context, params AlbumSearchParams) (*AlbumInfo, error)
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
