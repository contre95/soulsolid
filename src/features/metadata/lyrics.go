package metadata

import (
	"context"
)

// LyricsSearchParams contains parameters for searching lyrics
type LyricsSearchParams struct {
	TrackID     string
	AlbumArtist string
	Album       string
	Title       string
	Artist      string
}

// LyricsProvider defines the interface for fetching lyrics from external services
type LyricsProvider interface {
	// SearchLyrics searches for lyrics using metadata parameters and returns lyrics text
	SearchLyrics(ctx context.Context, params LyricsSearchParams) (string, error)

	// Name returns the provider name
	Name() string

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}
